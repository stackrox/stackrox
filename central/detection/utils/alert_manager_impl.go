package utils

import (
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
)

var (
	logger = logging.LoggerForModule()
)

type alertManagerImpl struct {
	notifier notifierProcessor.Processor
	alerts   alertDataStore.DataStore
}

func activeAlertsQueryBuilder() *search.QueryBuilder {
	return search.NewQueryBuilder().AddStrings(search.ViolationState, v1.ViolationState_ACTIVE.String())
}

func (d *alertManagerImpl) GetAlertsByLifecycle(lifecyle v1.LifecycleStage) ([]*v1.Alert, error) {
	q := activeAlertsQueryBuilder().
		AddStrings(search.LifecycleStage, lifecyle.String()).ProtoQuery()
	return d.alerts.SearchRawAlerts(q)
}

// GetAlertsByPolicy get all of the alerts that match the policy
func (d *alertManagerImpl) GetAlertsByPolicy(policyID string) ([]*v1.Alert, error) {
	qb := activeAlertsQueryBuilder().
		AddStrings(search.PolicyID, policyID)

	return d.alerts.SearchRawAlerts(qb.ProtoQuery())
}

// GetAlertsByDeployment get all of the alerts that match the deployment
func (d *alertManagerImpl) GetAlertsByDeployment(deploymentID string) ([]*v1.Alert, error) {
	qb := activeAlertsQueryBuilder().
		AddStrings(search.DeploymentID, deploymentID)

	return d.alerts.SearchRawAlerts(qb.ProtoQuery())
}

func (d *alertManagerImpl) GetAlertsByPolicyAndLifecycle(policyID string, lifecycle v1.LifecycleStage) ([]*v1.Alert, error) {
	q := activeAlertsQueryBuilder().
		AddStrings(search.PolicyID, policyID).
		AddStrings(search.LifecycleStage, lifecycle.String()).ProtoQuery()

	alerts, err := d.alerts.SearchRawAlerts(q)
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

func (d *alertManagerImpl) GetAlertsByDeploymentAndPolicyLifecycle(deploymentID string, lifecycle v1.LifecycleStage) ([]*v1.Alert, error) {
	q := activeAlertsQueryBuilder().
		AddStrings(search.DeploymentID, deploymentID).
		AddStrings(search.LifecycleStage, lifecycle.String()).ProtoQuery()

	alerts, err := d.alerts.SearchRawAlerts(q)
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// AlertAndNotify inserts and notifies of any new alerts (alerts in current but not in previous) deduplicated and
// updates those still produced (in both previous and current) and marks those no longer produced (in previous but
// not current) as stale.
func (d *alertManagerImpl) AlertAndNotify(previousAlerts, currentAlerts []*v1.Alert) error {
	// Merge the old and the new alerts.
	newAlerts, updatedAlerts, staleAlerts := d.mergeManyAlerts(previousAlerts, currentAlerts)

	// Mark any old alerts no longer generated as stale, and insert new alerts.
	if err := d.notifyAndUpdateBatch(newAlerts); err != nil {
		return err
	}
	if err := d.updateBatch(updatedAlerts); err != nil {
		return err
	}
	return d.markAlertsStale(staleAlerts)
}

// UpdateBatch updates all of the alerts in the datastore.
func (d *alertManagerImpl) updateBatch(alertsToMark []*v1.Alert) error {
	errList := errorhelpers.NewErrorList("Error updating alerts: ")
	for _, existingAlert := range alertsToMark {
		errList.AddError(d.alerts.UpdateAlert(existingAlert))
	}
	return errList.ToError()
}

// MarkAlertsStale marks all of the input alerts stale in the input datastore.
func (d *alertManagerImpl) markAlertsStale(alertsToMark []*v1.Alert) error {
	errList := errorhelpers.NewErrorList("Error marking alerts as stale: ")
	for _, existingAlert := range alertsToMark {
		errList.AddError(d.alerts.MarkAlertStale(existingAlert.GetId()))
	}
	return errList.ToError()
}

// NotifyAndUpdateBatch runs the notifier on the input alerts then stores them.
func (d *alertManagerImpl) notifyAndUpdateBatch(alertsToMark []*v1.Alert) error {
	for _, existingAlert := range alertsToMark {
		d.notifier.ProcessAlert(existingAlert)
	}
	return d.updateBatch(alertsToMark)
}

// We want to avoid continuously updating the timestamp of a runtime alert.
func equalRuntimeAlerts(old, new *v1.Alert) bool {
	if old.GetLifecycleStage() != v1.LifecycleStage_RUNTIME || new.GetLifecycleStage() != v1.LifecycleStage_RUNTIME {
		return false
	}
	if len(old.GetViolations()) != len(new.GetViolations()) {
		return false
	}
	for i, oldViolation := range old.GetViolations() {
		if !proto.Equal(oldViolation, new.GetViolations()[i]) {
			return false
		}
	}
	return true
}

// This depends on the fact that we sort process indicators in increasing order of timestamp.
func getMostRecentProcessTimestampInAlerts(alerts ...*v1.Alert) (ts *ptypes.Timestamp) {
	for _, alert := range alerts {
		for _, violation := range alert.GetViolations() {
			processes := violation.GetProcesses()
			if len(processes) == 0 {
				continue
			}
			lastTimeStampInViolation := processes[len(processes)-1].GetSignal().GetTime()
			if protoconv.CompareProtoTimestamps(ts, lastTimeStampInViolation) < 0 {
				ts = lastTimeStampInViolation
			}
		}
	}
	return
}

// If a user has resolved some process indicators from a particular alert, we don't want to display them
// if an alert was violated by a new indicator. This function trims old resolved processes from an alert.
// It returns a bool indicating whether the alert contained only resolved processes -- in which case
// we don't want to generate an alert at all.
func (d *alertManagerImpl) trimResolvedProcessesFromRuntimeAlert(alert *v1.Alert) (isFullyResolved bool) {
	if alert.GetLifecycleStage() != v1.LifecycleStage_RUNTIME {
		return false
	}

	q := search.NewQueryBuilder().
		AddStrings(search.ViolationState, v1.ViolationState_RESOLVED.String()).
		AddStrings(search.DeploymentID, alert.GetDeployment().GetId()).
		AddStrings(search.PolicyID, alert.GetPolicy().GetId()).
		ProtoQuery()

	oldRunTimeAlerts, err := d.alerts.SearchRawAlerts(q)
	// If there's an error, just log it, and assume there was no previously resolved alert.
	if err != nil {
		logger.Errorf("Failed to retrieve resolved runtime alerts corresponding to %+v: %s", alert, err)
		return false
	}
	if len(oldRunTimeAlerts) == 0 {
		return false
	}

	mostRecentResolvedTimestamp := getMostRecentProcessTimestampInAlerts(oldRunTimeAlerts...)

	var newProcessFound bool
	for _, violation := range alert.GetViolations() {
		if len(violation.GetProcesses()) == 0 {
			continue
		}
		filtered := violation.GetProcesses()[:0]
		for _, process := range violation.GetProcesses() {
			if protoconv.CompareProtoTimestamps(process.GetSignal().GetTime(), mostRecentResolvedTimestamp) > 0 {
				newProcessFound = true
				filtered = append(filtered, process)
			}
		}
		violation.Processes = filtered
	}
	isFullyResolved = !newProcessFound
	return
}

// MergeAlerts merges two alerts.
func mergeAlerts(old, new *v1.Alert) *v1.Alert {
	if equalRuntimeAlerts(old, new) {
		return old
	}
	new.Id = old.GetId()
	new.Enforcement = old.GetEnforcement()
	new.FirstOccurred = old.GetFirstOccurred()
	return new
}

// MergeManyAlerts merges two alerts.
func (d *alertManagerImpl) mergeManyAlerts(previousAlerts, presentAlerts []*v1.Alert) (newAlerts, updatedAlerts, staleAlerts []*v1.Alert) {
	// Merge any alerts that have new and old alerts.
	for _, alert := range presentAlerts {
		// Don't generate a new alert if it was a resolved runtime alerts.
		isFullyResolvedRuntimeAlert := d.trimResolvedProcessesFromRuntimeAlert(alert)
		if isFullyResolvedRuntimeAlert {
			continue
		}
		if matchingOld := findAlert(alert, previousAlerts); matchingOld != nil {
			updatedAlerts = append(updatedAlerts, mergeAlerts(matchingOld, alert))
			continue
		}

		alert.FirstOccurred = ptypes.TimestampNow()
		newAlerts = append(newAlerts, alert)
	}

	// Find any old alerts no longer being produced.
	for _, alert := range previousAlerts {
		if matchingNew := findAlert(alert, presentAlerts); matchingNew == nil {
			staleAlerts = append(staleAlerts, alert)
		}
	}
	return
}

func findAlert(toFind *v1.Alert, alerts []*v1.Alert) *v1.Alert {
	for _, alert := range alerts {
		if alertsAreForSamePolicyAndDeployment(alert, toFind) {
			return alert
		}
	}
	return nil
}

func alertsAreForSamePolicyAndDeployment(a1, a2 *v1.Alert) bool {
	return a1.GetPolicy().GetId() == a2.GetPolicy().GetId() && a1.GetDeployment().GetId() == a2.GetDeployment().GetId()
}
