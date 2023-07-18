package alertmanager

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/generated/storage"
	pkgAlert "github.com/stackrox/rox/pkg/alert"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	notifierProcessor "github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

const maxRunTimeViolationsPerAlert = 40

var (
	log = logging.LoggerForModule()
)

type alertManagerImpl struct {
	notifier        notifierProcessor.Processor
	alerts          alertDataStore.DataStore
	runtimeDetector runtime.Detector
}

// getDeploymentIDsFromAlerts returns a set of deployment IDs for given lists of alerts
func getDeploymentIDsFromAlerts(alertSlices ...[]*storage.Alert) set.StringSet {
	s := set.NewStringSet()
	for _, slice := range alertSlices {
		for _, alert := range slice {
			if dep := alert.GetDeployment(); dep != nil {
				s.Add(dep.GetId())
			}
		}
	}
	return s
}

// AlertAndNotify is the main function that implements the AlertManager interface
func (d *alertManagerImpl) AlertAndNotify(ctx context.Context, currentAlerts []*storage.Alert, oldAlertFilters ...AlertFilterOption) (set.StringSet, error) {
	// Merge the old and the new alerts.
	newAlerts, updatedAlerts, toBeResolvedAlerts, err := d.mergeManyAlerts(ctx, currentAlerts, oldAlertFilters...)
	if err != nil {
		return nil, err
	}

	// If any of the alerts are for a deployment, detect if the deployment itself is modified
	modifiedDeployments := getDeploymentIDsFromAlerts(newAlerts, updatedAlerts, toBeResolvedAlerts)

	// Mark any old alerts no longer generated as resolved, and insert new alerts.
	err = d.notifyAndUpdateBatch(ctx, newAlerts)
	if err != nil {
		return nil, err
	}
	err = d.updateBatch(ctx, updatedAlerts)
	if err != nil {
		return nil, err
	}
	err = d.markAlertsResolved(ctx, toBeResolvedAlerts)
	if err != nil {
		return nil, err
	}

	// rox-7233: runtime alerts always get notifications so send for updated alerts as well
	d.notifyUpdatedRuntimeAlerts(ctx, updatedAlerts)

	return modifiedDeployments, nil
}

// notifyUpdatedRuntimeAlerts sends alerts for all updated runtime events that occur
func (d *alertManagerImpl) notifyUpdatedRuntimeAlerts(ctx context.Context, updatedAlerts []*storage.Alert) {
	// this env var is for customer configurability to allow time to adapt to the new expected behavior
	//  planned to be removed in a future release (See ROX-8989)
	if !env.NotifyOnEveryRuntimeEvent() {
		return
	}

	for _, alert := range updatedAlerts {
		if alert.GetLifecycleStage() == storage.LifecycleStage_RUNTIME {
			d.notifier.ProcessAlert(ctx, alert)
		}
	}
}

// updateBatch updates all alerts in the datastore.
func (d *alertManagerImpl) updateBatch(ctx context.Context, alertsToMark []*storage.Alert) error {
	errList := errorhelpers.NewErrorList("Error updating alerts: ")
	for _, existingAlert := range alertsToMark {
		errList.AddError(d.alerts.UpsertAlert(ctx, existingAlert))
	}
	return errList.ToError()
}

// markAlertsResolved marks all input alerts resolved in the input datastore.
func (d *alertManagerImpl) markAlertsResolved(ctx context.Context, alertsToMark []*storage.Alert) error {
	if len(alertsToMark) == 0 {
		return nil
	}

	ids := make([]string, 0, len(alertsToMark))
	for _, alert := range alertsToMark {
		ids = append(ids, alert.GetId())
	}
	resolvedAlerts, err := d.alerts.MarkAlertsResolvedBatch(ctx, ids...)
	if err != nil {
		return err
	}
	for _, resolvedAlert := range resolvedAlerts {
		d.notifier.ProcessAlert(ctx, resolvedAlert)
	}
	return nil
}

func (d *alertManagerImpl) shouldDebounceNotification(ctx context.Context, alert *storage.Alert) bool {
	if alert.GetLifecycleStage() != storage.LifecycleStage_DEPLOY {
		return false
	}
	dur := env.AlertRenotifDebounceDuration.DurationSetting()
	if dur == 0 {
		return false
	}

	maxAllowedResolvedAtTime, err := ptypes.TimestampProto(time.Now().Add(-dur))
	if err != nil {
		log.Errorf("Failed to convert time: %v", err)
		return false
	}
	q := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, alert.GetDeployment().GetId()).
		AddExactMatches(search.PolicyID, alert.GetPolicy().GetId()).
		AddExactMatches(search.ViolationState, storage.ViolationState_RESOLVED.String()).
		ProtoQuery()
	resolvedAlerts, err := d.alerts.SearchRawAlerts(ctx, q)
	if err != nil {
		log.Errorf("Error fetching formerly resolved alerts for alert %s: %v", alert.GetId(), err)
		return false
	}
	for _, resolvedAlert := range resolvedAlerts {
		resolvedAt := resolvedAlert.GetResolvedAt()
		// This alert was resolved very recently, so debounce the notification.
		if resolvedAt != nil && resolvedAt.Compare(maxAllowedResolvedAtTime) > 0 {
			return true
		}
	}
	return false
}

// notifyAndUpdateBatch runs the notifier on the input alerts then stores them.
func (d *alertManagerImpl) notifyAndUpdateBatch(ctx context.Context, alertsToMark []*storage.Alert) error {
	for _, existingAlert := range alertsToMark {
		if d.shouldDebounceNotification(ctx, existingAlert) {
			continue
		}
		d.notifier.ProcessAlert(ctx, existingAlert)
	}
	return d.updateBatch(ctx, alertsToMark)
}

// It is the caller's responsibility to not call this with an empty slice.
func lastTimestamp(processes []*storage.ProcessIndicator) (*ptypes.Timestamp, error) {
	if len(processes) == 0 {
		return nil, errors.New("Unexpected: no processes found in the alert")
	}
	return processes[len(processes)-1].GetSignal().GetTime(), nil
}

// Some processes in the old alert might have been deleted from the process store because of our pruning,
// which means they only exist in the old alert, and will not be in the new generated alert.
// We don't want to lose them, though, so we keep all the processes from the old alert, and add ones from the new, if any.
// Note that the old alert _was_ active which means that all the processes in it are guaranteed to violate the policy.
func mergeProcessesFromOldIntoNew(old, newAlert *storage.Alert) (newAlertHasNewProcesses bool) {
	oldProcessViolation := old.GetProcessViolation()

	// Do not return if the old alert has 0 processes because that is unexpected. Further down we log the error.
	if len(newAlert.GetProcessViolation().GetProcesses()) == 0 {
		return
	}

	if len(oldProcessViolation.GetProcesses()) >= maxRunTimeViolationsPerAlert {
		return
	}

	newProcessesSlice := oldProcessViolation.GetProcesses()
	// De-dupe processes using timestamps.
	timestamp, err := lastTimestamp(oldProcessViolation.GetProcesses())
	if err != nil {
		log.Errorf(
			"Failed to merge alerts. "+
				"New alert %s (policy=%s) has %d processses and old alert %s (policy=%s) has %d processes: %v",
			newAlert.GetId(), newAlert.GetPolicy().GetName(), len(newAlert.GetProcessViolation().GetProcesses()),
			old.GetId(), old.GetPolicy().GetName(), len(oldProcessViolation.GetProcesses()), err,
		)
		// At this point, we know that the new alert has non-zero process violations but it cannot be merged.
		newAlertHasNewProcesses = true
		return
	}

	for _, process := range newAlert.GetProcessViolation().GetProcesses() {
		if process.GetSignal().GetTime().Compare(timestamp) > 0 {
			newAlertHasNewProcesses = true
			newProcessesSlice = append(newProcessesSlice, process)
		}
	}
	// If there are no new processes, we'll just use the old alert.
	if !newAlertHasNewProcesses {
		return
	}
	if len(newProcessesSlice) > maxRunTimeViolationsPerAlert {
		newProcessesSlice = newProcessesSlice[:maxRunTimeViolationsPerAlert]
	}
	newAlert.ProcessViolation.Processes = newProcessesSlice
	printer.UpdateProcessAlertViolationMessage(newAlert.ProcessViolation)
	return
}

// Since we are only generating one alert per flow instance (we could be generating multiple violations on the same
// "flow", but those are still for different flow instances), we can just sort by latest first.
func mergeNetworkFlowViolations(old, new *storage.Alert) bool {
	return mergeAlertsByLatestFirst(old, new, storage.Alert_Violation_NETWORK_FLOW)
}

// mergeRunTimeAlerts merges run-time alerts, and returns true if new alert has at least one new run-time violation.
func mergeRunTimeAlerts(old, newAlert *storage.Alert) bool {
	newAlertHasNewProcesses := mergeProcessesFromOldIntoNew(old, newAlert)
	newAlertHasNewEventViolations := mergeK8sEventViolations(old, newAlert)
	newAlertHasNewNetworkFlowViolations := mergeNetworkFlowViolations(old, newAlert)
	return newAlertHasNewProcesses || newAlertHasNewEventViolations || newAlertHasNewNetworkFlowViolations
}

// Given the nature of an event, each event it anticipated to generate exactly one alert (one or more violations).
// Therefore, event violations seen in new alerts are assumed to be distinct from the old.
// For k8s event violations we want to *always* show the recent events. This approach is different from the way process
// violations are dealt where longest running processes take precedence over new processes.
func mergeK8sEventViolations(old, new *storage.Alert) bool {
	return mergeAlertsByLatestFirst(old, new, storage.Alert_Violation_K8S_EVENT)
}

// mergeAlertsByLatestFirst is for alert violations that are NOT aggregated under one drop-down.
func mergeAlertsByLatestFirst(old, new *storage.Alert, alertType storage.Alert_Violation_Type) bool {
	var newViolations []*storage.Alert_Violation
	for _, v := range new.GetViolations() {
		if v.GetType() == alertType {
			newViolations = append(newViolations, v)
		}
	}

	if len(newViolations) == 0 {
		return false
	}

	// New alert takes precedence. Do not merge any old event violations into new alert if we are already at threshold.
	if len(newViolations) >= maxRunTimeViolationsPerAlert {
		return true
	}

	var oldViolations []*storage.Alert_Violation
	// Append old violations to the end of the list so that they appear at bottom in UI.
	for _, v := range old.GetViolations() {
		if v.GetType() == alertType {
			oldViolations = append(oldViolations, v)
		}
	}

	newViolations = append(newViolations, oldViolations...)

	if len(newViolations) > maxRunTimeViolationsPerAlert {
		newViolations = newViolations[:maxRunTimeViolationsPerAlert]
	}
	new.Violations = newViolations
	// Since violations are not aggregated under one drop-down, no other message changes required.

	return true
}

// mergeAlerts merges two alerts. The caller to this ensures that both alerts match (i.e. they are either for the same policy & deployment OR same policy & resource.)
func mergeAlerts(old, newAlert *storage.Alert) *storage.Alert {
	if old.GetLifecycleStage() == storage.LifecycleStage_RUNTIME && newAlert.GetLifecycleStage() == storage.LifecycleStage_RUNTIME {
		// This ensures that we don't keep updating an old runtime alert, so that we have idempotent checks.
		if newAlertHasNewRuntimeViolations := mergeRunTimeAlerts(old, newAlert); !newAlertHasNewRuntimeViolations {
			return old
		}
	}

	newAlert.Id = old.GetId()
	// Updated deploy-time alerts continue to have the same enforcement action.
	if newAlert.GetLifecycleStage() == storage.LifecycleStage_DEPLOY && old.GetLifecycleStage() == storage.LifecycleStage_DEPLOY {
		newAlert.Enforcement = old.GetEnforcement()
		// Don't keep updating the timestamp of the violation _unless_ the violations are actually different.
		if protoutils.SlicesEqual(newAlert.GetViolations(), old.GetViolations()) {
			newAlert.Time = old.GetTime()
		}
	}

	newAlert.FirstOccurred = old.GetFirstOccurred()
	return newAlert
}

// mergeManyAlerts merges two alerts.
func (d *alertManagerImpl) mergeManyAlerts(
	ctx context.Context,
	incomingAlerts []*storage.Alert,
	oldAlertFilters ...AlertFilterOption,
) (newAlerts, updatedAlerts, toBeResolvedAlerts []*storage.Alert, err error) {
	qb := search.NewQueryBuilder().AddExactMatches(
		search.ViolationState,
		storage.ViolationState_ACTIVE.String(),
		storage.ViolationState_ATTEMPTED.String())
	for _, filter := range oldAlertFilters {
		filter.apply(qb)
	}
	previousAlerts, err := d.alerts.SearchRawAlerts(ctx, qb.ProtoQuery())
	if err != nil {
		err = errors.Wrapf(err, "couldn't load previous alerts (query was %s)", qb.Query())
		return
	}

	// Merge any alerts that have new and old alerts.
	for _, alert := range incomingAlerts {
		if pkgAlert.IsDeployTimeAttemptedAlert(alert) {
			// `alert.time` is the latest violation time.
			alert.FirstOccurred = alert.GetTime()
			newAlerts = append(newAlerts, alert)
			continue
		}

		if matchingOld := findAlert(alert, previousAlerts); matchingOld != nil {
			mergedAlert := mergeAlerts(matchingOld, alert)
			if mergedAlert != matchingOld && !proto.Equal(mergedAlert, matchingOld) {
				updatedAlerts = append(updatedAlerts, mergedAlert)
			}
			continue
		}

		// `alert.time` is the latest violation time.
		alert.FirstOccurred = alert.GetTime()
		newAlerts = append(newAlerts, alert)
	}

	// Get the deployments that are currently being removed as part of this alert update.
	deploymentsBeingRemoved := set.NewStringSet()
	for _, f := range oldAlertFilters {
		if depID := f.removedDeploymentID(); depID != "" {
			deploymentsBeingRemoved.Add(depID)
		}
	}

	// Find any old alerts no longer being produced.
	for _, previousAlert := range previousAlerts {
		if d.shouldMarkAlertResolved(previousAlert, incomingAlerts, oldAlertFilters...) {
			toBeResolvedAlerts = append(toBeResolvedAlerts, previousAlert)
		}

		if previousAlert.GetLifecycleStage() == storage.LifecycleStage_RUNTIME ||
			previousAlert.GetState() == storage.ViolationState_ATTEMPTED {
			// If we are in the context of a deployment removal, then mark the deployment as inactive without going to the
			// store -- this is because the alert processing related to the deployment removal (ie, this code) runs
			// _before_ the deployment is actually deleted, which means we will incorrectly mark the deployment as active
			// if we check the store.
			// If we're not in the context of a deployment removal, then just check whether the deployment is inactive
			// in the store and use that.
			if deployment := previousAlert.GetDeployment(); deployment != nil {
				depID := deployment.GetId()
				if deploymentsBeingRemoved.Contains(depID) || d.runtimeDetector.DeploymentInactive(depID) {
					if deployment := previousAlert.GetDeployment(); deployment != nil && !deployment.GetInactive() {
						deployment.Inactive = true
						updatedAlerts = append(updatedAlerts, previousAlert)
					}
				}
			}
		}
	}
	return
}

func (d *alertManagerImpl) shouldMarkAlertResolved(oldAlert *storage.Alert, incomingAlerts []*storage.Alert, oldAlertFilters ...AlertFilterOption) bool {
	oldAndNew := []*storage.Alert{oldAlert}
	oldAndNew = append(oldAndNew, incomingAlerts...)
	// Do not mark any attempted alerts as stale. All attempted alerts must be resolved by users.
	if pkgAlert.AnyAttemptedAlert(oldAndNew...) {
		return false
	}

	// If the alert is still being produced, don't mark it stale.
	if matchingNew := findAlert(oldAlert, incomingAlerts); matchingNew != nil {
		return false
	}

	// Only runtime alerts should not be marked stale when they are no longer produced.
	// (Deploy time alerts should disappear along with deployments, for example.)
	if oldAlert.GetLifecycleStage() != storage.LifecycleStage_RUNTIME {
		return true
	}

	if !d.runtimeDetector.PolicySet().Exists(oldAlert.GetPolicy().GetId()) {
		return true
	}

	// We only want to mark runtime alerts as stale if a policy update causes them to no longer be produced.
	// To determine if this is a policy update, we check if there is a filter on policy ids here.
	specifiedPolicyIDs := set.NewStringSet()
	for _, filter := range oldAlertFilters {
		if filterSpecified := filter.specifiedPolicyID(); filterSpecified != "" {
			specifiedPolicyIDs.Add(filterSpecified)
		}
	}
	if specifiedPolicyIDs.Cardinality() == 0 {
		return false
	}

	// Some other policies were updated, we don't want to mark this alert stale in response.
	if !specifiedPolicyIDs.Contains(oldAlert.GetPolicy().GetId()) {
		return false
	}

	// If the deployment is excluded from the scope of the policy now, we should mark the alert stale, otherwise we will keep it around.
	return d.runtimeDetector.DeploymentWhitelistedForPolicy(oldAlert.GetDeployment().GetId(), oldAlert.GetPolicy().GetId())
}

func findAlert(toFind *storage.Alert, alerts []*storage.Alert) *storage.Alert {
	for _, alert := range alerts {
		if alertsAreForSamePolicyAndEntity(alert, toFind) {
			return alert
		}
	}
	return nil
}

func alertsAreForSamePolicyAndEntity(a1, a2 *storage.Alert) bool {
	if a1.GetPolicy().GetId() != a2.GetPolicy().GetId() || a1.GetState() != a2.GetState() {
		return false
	}

	if a1.GetDeployment() != nil && a2.GetDeployment() != nil {
		return a1.GetDeployment().GetId() == a2.GetDeployment().GetId()
	} else if a1.GetResource() != nil && a2.GetResource() != nil {
		return alertsAreForSameResource(a1.GetResource(), a2.GetResource())
	}
	return false
}

func alertsAreForSameResource(a1, a2 *storage.Alert_Resource) bool {
	return a1.GetResourceType() == a2.GetResourceType() &&
		a1.GetName() == a2.GetName() &&
		a1.GetClusterId() == a2.GetClusterId() &&
		a1.GetNamespace() == a2.GetNamespace()
}
