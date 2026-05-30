package alertmanager

import (
	"context"
	"time"

	"github.com/pkg/errors"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	alertviews "github.com/stackrox/rox/central/alert/views"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/generated/storage"
	pkgAlert "github.com/stackrox/rox/pkg/alert"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	notifierProcessor "github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
)

const maxRunTimeViolationsPerAlert = 40

var (
	log = logging.LoggerForModule()
)

type mergeCandidate struct {
	incoming *storage.Alert
	oldKeyID string
}

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
	defer observeDurationMs(alertAndNotifyDuration)()
	alertAndNotifyIncomingCount.Observe(float64(len(currentAlerts)))

	// Merge the old and the new alerts.
	newAlerts, updatedAlerts, toBeResolvedIDs, resolvedDeploymentIDs, err := d.mergeManyAlerts(ctx, currentAlerts, oldAlertFilters...)
	if err != nil {
		return nil, err
	}

	// If any of the alerts are for a deployment, detect if the deployment itself is modified
	modifiedDeployments := getDeploymentIDsFromAlerts(newAlerts, updatedAlerts)
	modifiedDeployments = modifiedDeployments.Union(resolvedDeploymentIDs)

	// Mark any old alerts no longer generated as resolved, and insert new alerts.
	err = d.notifyAndUpdateBatch(ctx, newAlerts)
	if err != nil {
		return nil, err
	}
	err = d.updateBatch(ctx, updatedAlerts)
	if err != nil {
		return nil, err
	}
	err = d.markAlertsResolvedByIDs(ctx, toBeResolvedIDs)
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

// markAlertsResolvedByIDs marks alerts with the given IDs as resolved.
func (d *alertManagerImpl) markAlertsResolvedByIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
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

	maxAllowedResolvedAtTime, err := protocompat.ConvertTimeToTimestampOrError(time.Now().Add(-dur))
	if err != nil {
		log.Errorf("Failed to convert time: %v", err)
		return false
	}
	q := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, alert.GetDeployment().GetId()).
		AddExactMatches(search.PolicyID, alert.GetPolicy().GetId()).
		AddExactMatches(search.ViolationState, storage.ViolationState_RESOLVED.String()).
		ProtoQuery()
	resolvedAlerts, err := d.alerts.SearchRawAlerts(ctx, q, false)
	if err != nil {
		log.Errorf("Error fetching formerly resolved alerts for alert %s: %v", alert.GetId(), err)
		return false
	}
	for _, resolvedAlert := range resolvedAlerts {
		resolvedAt := resolvedAlert.GetResolvedAt()
		// This alert was resolved very recently, so debounce the notification.
		if resolvedAt != nil && protocompat.CompareTimestamps(resolvedAt, maxAllowedResolvedAtTime) > 0 {
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
func lastTime(processes []*storage.ProcessIndicator) (*time.Time, error) {
	if len(processes) == 0 {
		return nil, errors.New("Unexpected: no processes found in the alert")
	}
	lastTime := protocompat.ConvertTimestampToTimeOrNil(processes[len(processes)-1].GetSignal().GetTime())
	return lastTime, nil
}

func lastFileTime(violations []*storage.Alert_Violation) (*time.Time, error) {
	if len(violations) == 0 {
		return nil, errors.New("Unexpected: no file access violations found in the alert")
	}
	lastFileAccess := violations[len(violations)-1].GetFileAccess()
	if lastFileAccess == nil {
		return nil, errors.New("Unexpected: file access violation missing file access data")
	}
	lastTime := protocompat.ConvertTimestampToTimeOrNil(lastFileAccess.GetTimestamp())
	return lastTime, nil
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
	lastProcessTime, err := lastTime(oldProcessViolation.GetProcesses())
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
		if protocompat.CompareTimestampToTime(process.GetSignal().GetTime(), lastProcessTime) > 0 {
			newAlertHasNewProcesses = true
			newProcessesSlice = append(newProcessesSlice, process)
		}
	}
	// If there are no new processes, we'll just use the old alert.
	if !newAlertHasNewProcesses {
		return
	}
	if len(newProcessesSlice) > maxRunTimeViolationsPerAlert {
		// prioritize newer events over old ones
		newProcessesSlice = newProcessesSlice[len(newProcessesSlice)-maxRunTimeViolationsPerAlert:]
	}
	newAlert.ProcessViolation.Processes = newProcessesSlice
	printer.UpdateProcessAlertViolationMessage(newAlert.GetProcessViolation())
	return
}

// Since we are only generating one alert per flow instance (we could be generating multiple violations on the same
// "flow", but those are still for different flow instances), we can just sort by latest first.
func mergeNetworkFlowViolations(old, new *storage.Alert) bool {
	return mergeAlertsByLatestFirst(old, new, storage.Alert_Violation_NETWORK_FLOW)
}

func mergeFileAccessViolations(oldAlert, newAlert *storage.Alert) bool {
	// Extract FILE_ACCESS violations from both alerts
	newViolations := sliceutils.Filter(newAlert.GetViolations(), func(v *storage.Alert_Violation) bool {
		return v.GetType() == storage.Alert_Violation_FILE_ACCESS
	})

	oldViolations := sliceutils.Filter(oldAlert.GetViolations(), func(v *storage.Alert_Violation) bool {
		return v.GetType() == storage.Alert_Violation_FILE_ACCESS
	})

	if len(newViolations) == 0 || len(oldViolations) >= maxRunTimeViolationsPerAlert {
		return false
	}

	// Start with old violations and merge with new
	mergedViolations := oldViolations
	lastAccessTime, err := lastFileTime(oldViolations)
	if err != nil {
		log.Errorf(
			"Failed to merge alerts. "+
				"New alert %s (policy=%s) has %d file access violations and old alert %s (policy=%s) has %d file access violations: %v",
			newAlert.GetId(), newAlert.GetPolicy().GetName(), len(newViolations),
			oldAlert.GetId(), oldAlert.GetPolicy().GetName(), len(oldViolations), err,
		)
		// At this point, we know that the new alert has non-zero file violations but it cannot be merged.
		return true
	}

	hasNewAccesses := false
	for _, violation := range newViolations {
		fileAccess := violation.GetFileAccess()
		if fileAccess != nil && protocompat.CompareTimestampToTime(fileAccess.GetTimestamp(), lastAccessTime) > 0 {
			hasNewAccesses = true
			mergedViolations = append(mergedViolations, violation)
		}
	}

	// If there are no new accesses, we'll just use the old alert.
	if !hasNewAccesses {
		return false
	}

	if len(mergedViolations) > maxRunTimeViolationsPerAlert {
		// prioritize newer events over old ones
		mergedViolations = mergedViolations[len(mergedViolations)-maxRunTimeViolationsPerAlert:]
	}

	newAlert.Violations = mergedViolations
	return true
}

// mergeRunTimeAlerts merges run-time alerts, and returns true if new alert has at least one new run-time violation.
func mergeRunTimeAlerts(old, newAlert *storage.Alert) bool {
	newAlertHasNewProcesses := mergeProcessesFromOldIntoNew(old, newAlert)
	newAlertHasNewEventViolations := mergeK8sEventViolations(old, newAlert)
	newAlertHasNewNetworkFlowViolations := mergeNetworkFlowViolations(old, newAlert)
	newAlertHasNewFileAccessViolations := mergeFileAccessViolations(old, newAlert)
	return newAlertHasNewProcesses || newAlertHasNewEventViolations || newAlertHasNewNetworkFlowViolations || newAlertHasNewFileAccessViolations
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

// mergeManyAlerts uses a two-phase fetch to match incoming alerts against
// previous alerts. Phase 1 fetches lightweight AlertMatchKey structs (inline
// columns only, no TOAST I/O). Phase 2 fetches full alert blobs only for
// the small fraction that actually matched and need merging.
func (d *alertManagerImpl) mergeManyAlerts(
	ctx context.Context,
	incomingAlerts []*storage.Alert,
	oldAlertFilters ...AlertFilterOption,
) (newAlerts, updatedAlerts []*storage.Alert, toBeResolvedIDs []string, resolvedDeploymentIDs set.StringSet, err error) {
	defer observeDurationMs(mergeManyAlertsDuration)()
	resolvedDeploymentIDs = set.NewStringSet()

	qb := search.NewQueryBuilder().AddExactMatches(
		search.ViolationState,
		storage.ViolationState_ACTIVE.String(),
		storage.ViolationState_ATTEMPTED.String())
	for _, filter := range oldAlertFilters {
		filter.apply(qb)
	}

	// Phase 1: fetch lightweight match keys instead of full alert blobs.
	previousKeys, err := d.alerts.SearchAlertMatchKeys(ctx, qb.ProtoQuery(), true)
	if err != nil {
		err = errors.Wrapf(err, "couldn't load previous alert keys (query was %s)", qb.Query())
		return
	}
	mergeManyAlertsPreviousCount.Observe(float64(len(previousKeys)))

	// Match incoming alerts against previous keys, collecting merge candidates.
	var mergeCandidates []mergeCandidate

	for _, alert := range incomingAlerts {
		if pkgAlert.IsDeployTimeAttemptedAlert(alert) {
			// `alert.time` is the latest violation time.
			alert.FirstOccurred = alert.GetTime()
			newAlerts = append(newAlerts, alert)
			continue
		}

		if matchingKey := findMatchingKey(alert, previousKeys); matchingKey != nil {
			mergeCandidates = append(mergeCandidates, mergeCandidate{incoming: alert, oldKeyID: matchingKey.GetId()})
			continue
		}

		// `alert.time` is the latest violation time.
		alert.FirstOccurred = alert.GetTime()
		newAlerts = append(newAlerts, alert)
	}

	// Phase 2: fetch full alerts for matched keys and merge.
	mergedAlertIDs, mergedNew, mergedUpdated, mergeErr := d.fetchAndMergeCandidates(ctx, mergeCandidates)
	if mergeErr != nil {
		err = mergeErr
		return
	}
	newAlerts = append(newAlerts, mergedNew...)
	updatedAlerts = append(updatedAlerts, mergedUpdated...)

	// Find old alerts no longer being produced, and identify inactive deployments.
	deploymentsBeingRemoved := set.NewStringSet()
	for _, f := range oldAlertFilters {
		if depID := f.removedDeploymentID(); depID != "" {
			deploymentsBeingRemoved.Add(depID)
		}
	}

	var needInactiveIDs []string
	for _, key := range previousKeys {
		if d.shouldMarkAlertResolved(key, incomingAlerts, oldAlertFilters...) {
			toBeResolvedIDs = append(toBeResolvedIDs, key.GetId())
			if key.HasDeployment() {
				resolvedDeploymentIDs.Add(key.GetDeploymentId())
			}
		}

		if key.GetLifecycleStage() == storage.LifecycleStage_RUNTIME ||
			key.GetState() == storage.ViolationState_ATTEMPTED {
			if mergedAlertIDs.Contains(key.GetId()) {
				continue
			}
			if key.HasDeployment() && !key.IsDeploymentInactive() {
				depID := key.GetDeploymentId()
				if deploymentsBeingRemoved.Contains(depID) || d.runtimeDetector.DeploymentInactive(depID) {
					needInactiveIDs = append(needInactiveIDs, key.GetId())
				}
			}
		}
	}

	// Mark deployments inactive for alerts not already handled by the merge phase.
	inactiveUpdates, inactiveErr := d.markDeploymentsInactive(ctx, needInactiveIDs)
	if inactiveErr != nil {
		err = inactiveErr
		return
	}
	updatedAlerts = append(updatedAlerts, inactiveUpdates...)

	recordAlertOutcomes(len(newAlerts), len(updatedAlerts), len(toBeResolvedIDs))
	return
}

// fetchAndMergeCandidates fetches full alerts for merge candidates and merges
// them with incoming alerts. Returns the set of merged alert IDs (to skip in
// inactive marking), any alerts treated as new (deleted between phases), and
// any updated alerts from merging.
func (d *alertManagerImpl) fetchAndMergeCandidates(ctx context.Context, candidates []mergeCandidate) (mergedIDs set.StringSet, newAlerts, updatedAlerts []*storage.Alert, err error) {
	mergedIDs = set.NewStringSet()
	if len(candidates) == 0 {
		return
	}

	uniqueIDs := set.NewStringSet()
	for _, mc := range candidates {
		uniqueIDs.Add(mc.oldKeyID)
	}
	mergeQuery := search.NewQueryBuilder().
		AddExactMatches(search.AlertID, uniqueIDs.AsSlice()...).
		ProtoQuery()
	fetched, fetchErr := d.alerts.SearchRawAlerts(ctx, mergeQuery, false)
	if fetchErr != nil {
		err = errors.Wrap(fetchErr, "failed to fetch alerts for merge")
		return
	}
	oldAlertsByID := make(map[string]*storage.Alert, len(fetched))
	for _, a := range fetched {
		oldAlertsByID[a.GetId()] = a
	}

	for _, mc := range candidates {
		matchingOld := oldAlertsByID[mc.oldKeyID]
		if matchingOld == nil {
			mc.incoming.FirstOccurred = mc.incoming.GetTime()
			newAlerts = append(newAlerts, mc.incoming)
			continue
		}
		mergedAlert := mergeAlerts(matchingOld, mc.incoming)
		if !mergedAlert.EqualVT(matchingOld) {
			updatedAlerts = append(updatedAlerts, mergedAlert)
		}
		mergedIDs.Add(mc.oldKeyID)
	}
	return
}

// markDeploymentsInactive fetches alerts by ID and sets deployment.Inactive = true
// for any that have an active deployment.
func (d *alertManagerImpl) markDeploymentsInactive(ctx context.Context, alertIDs []string) ([]*storage.Alert, error) {
	if len(alertIDs) == 0 {
		return nil, nil
	}

	inactiveQuery := search.NewQueryBuilder().
		AddExactMatches(search.AlertID, alertIDs...).
		ProtoQuery()
	inactiveAlerts, err := d.alerts.SearchRawAlerts(ctx, inactiveQuery, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch alerts for inactive marking")
	}

	var updated []*storage.Alert
	for _, fullAlert := range inactiveAlerts {
		if deployment := fullAlert.GetDeployment(); deployment != nil && !deployment.GetInactive() {
			deployment.Inactive = true
			updated = append(updated, fullAlert)
		}
	}
	return updated, nil
}

func (d *alertManagerImpl) shouldMarkAlertResolved(old alertviews.AlertMatcher, incomingAlerts []*storage.Alert, oldAlertFilters ...AlertFilterOption) bool {
	// Do not mark any attempted alerts as stale. All attempted alerts must be resolved by users.
	if old.GetState() == storage.ViolationState_ATTEMPTED {
		return false
	}
	for _, incoming := range incomingAlerts {
		if incoming.GetState() == storage.ViolationState_ATTEMPTED {
			return false
		}
	}

	// If the alert is still being produced, don't mark it stale.
	if matchingNew := findMatchingAlert(old, incomingAlerts); matchingNew != nil {
		return false
	}

	// Only runtime alerts should not be marked stale when they are no longer produced.
	// (Deploy time alerts should disappear along with deployments, for example.)
	if old.GetLifecycleStage() != storage.LifecycleStage_RUNTIME {
		return true
	}

	if !d.runtimeDetector.PolicySet().Exists(old.GetPolicyId()) {
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
	if !specifiedPolicyIDs.Contains(old.GetPolicyId()) {
		return false
	}

	// If the deployment is excluded from the scope of the policy now, we should mark the alert stale, otherwise we will keep it around.
	return d.runtimeDetector.DeploymentWhitelistedForPolicy(old.GetDeploymentId(), old.GetPolicyId())
}

// alertAdapter wraps *storage.Alert to satisfy the AlertMatcher interface.
type alertAdapter struct{ a *storage.Alert }

func (w alertAdapter) GetId() string                             { return w.a.GetId() }
func (w alertAdapter) GetPolicyId() string                       { return w.a.GetPolicy().GetId() }
func (w alertAdapter) GetState() storage.ViolationState          { return w.a.GetState() }
func (w alertAdapter) GetLifecycleStage() storage.LifecycleStage { return w.a.GetLifecycleStage() }
func (w alertAdapter) HasDeployment() bool                       { return w.a.GetDeployment() != nil }
func (w alertAdapter) GetDeploymentId() string                   { return w.a.GetDeployment().GetId() }
func (w alertAdapter) IsDeploymentInactive() bool                { return w.a.GetDeployment().GetInactive() }
func (w alertAdapter) HasResource() bool                         { return w.a.GetResource() != nil }
func (w alertAdapter) GetResourceType() storage.Alert_Resource_ResourceType {
	return w.a.GetResource().GetResourceType()
}
func (w alertAdapter) GetResourceName() string { return w.a.GetResource().GetName() }
func (w alertAdapter) HasNode() bool           { return w.a.GetNode() != nil }
func (w alertAdapter) GetNodeId() string       { return w.a.GetNode().GetId() }
func (w alertAdapter) GetNodeName() string     { return w.a.GetNode().GetName() }

// GetClusterId returns the entity-specific cluster ID to match the original
// comparison behavior. Resource and node alerts store the cluster ID on their
// sub-objects; the top-level Alert.ClusterId is a denormalized copy that may
// not be set in unit test fixtures.
func (w alertAdapter) GetClusterId() string {
	if r := w.a.GetResource(); r != nil {
		return r.GetClusterId()
	}
	if n := w.a.GetNode(); n != nil {
		return n.GetClusterId()
	}
	return w.a.GetClusterId()
}

// GetNamespace returns the entity-specific namespace. Resource alerts store the
// namespace on Alert_Resource; deployment alerts use the top-level field.
func (w alertAdapter) GetNamespace() string {
	if r := w.a.GetResource(); r != nil {
		return r.GetNamespace()
	}
	return w.a.GetNamespace()
}

func findMatchingAlert(toFind alertviews.AlertMatcher, alerts []*storage.Alert) *storage.Alert {
	for _, alert := range alerts {
		if alertsAreForSamePolicyAndEntity(toFind, alertAdapter{alert}) {
			return alert
		}
	}
	return nil
}

func findMatchingKey(toFind *storage.Alert, keys []*alertviews.AlertMatchKey) *alertviews.AlertMatchKey {
	wrapper := alertAdapter{toFind}
	for _, key := range keys {
		if alertsAreForSamePolicyAndEntity(wrapper, key) {
			return key
		}
	}
	return nil
}

func alertsAreForSamePolicyAndEntity(a1, a2 alertviews.AlertMatcher) bool {
	if a1.GetPolicyId() != a2.GetPolicyId() || a1.GetState() != a2.GetState() {
		return false
	}

	if a1.HasDeployment() && a2.HasDeployment() {
		return a1.GetDeploymentId() == a2.GetDeploymentId()
	} else if a1.HasResource() && a2.HasResource() {
		return a1.GetResourceType() == a2.GetResourceType() &&
			a1.GetResourceName() == a2.GetResourceName() &&
			a1.GetClusterId() == a2.GetClusterId() &&
			a1.GetNamespace() == a2.GetNamespace()
	} else if a1.HasNode() && a2.HasNode() {
		return a1.GetNodeId() == a2.GetNodeId() &&
			a1.GetNodeName() == a2.GetNodeName() &&
			a1.GetClusterId() == a2.GetClusterId()
	}
	return false
}
