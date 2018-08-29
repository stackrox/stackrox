package deploytime

import (
	ptypes "github.com/gogo/protobuf/types"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/search"
)

type alertManagerImpl struct {
	notifier notifierProcessor.Processor
	alerts   alertDataStore.DataStore
}

// GetAlertsByPolicy get all of the alerts that match the policy
func (d *alertManagerImpl) GetAlertsByPolicy(policyID string) ([]*v1.Alert, error) {
	qb := search.NewQueryBuilder().
		AddBools(search.Stale, false).
		AddStrings(search.PolicyID, policyID)

	return d.alerts.SearchRawAlerts(qb.ProtoQuery())
}

// GetAlertsByDeployment get all of the alerts that match the deployment
func (d *alertManagerImpl) GetAlertsByDeployment(deploymentID string) ([]*v1.Alert, error) {
	qb := search.NewQueryBuilder().
		AddBools(search.Stale, false).
		AddStrings(search.DeploymentID, deploymentID)

	return d.alerts.SearchRawAlerts(qb.ProtoQuery())
}

// AlertAndNotify inserts and notifies of any new alerts (alerts in current but not in previous) deduplicated and
// updates those still produced (in both previous and current) and marks those no longer produced (in previous but
// not current) as stale.
func (d *alertManagerImpl) AlertAndNotify(previousAlerts, currentAlerts []*v1.Alert) error {
	// Merge the old and the new alerts.
	newAlerts, updatedAlerts, staleAlerts := mergeManyAlerts(previousAlerts, currentAlerts)

	// Mark any old alerts no longer generated as stale, and insert new alerts.
	if err := d.notifyAndUpdateBatch(newAlerts); err != nil {
		return err
	}
	if err := d.updateBatch(updatedAlerts); err != nil {
		return err
	}
	if err := d.markAlertsStale(staleAlerts); err != nil {
		return err
	}
	return nil
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
		existingAlert.Stale = true
		existingAlert.MarkedStale = ptypes.TimestampNow()
		errList.AddError(d.alerts.UpdateAlert(existingAlert))
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

// MergeAlerts merges two alerts.
func mergeAlerts(old, new *v1.Alert) *v1.Alert {
	new.Id = old.GetId()
	new.Enforcement = old.GetEnforcement()
	new.FirstOccurred = old.GetFirstOccurred()
	return new
}

// MergeManyAlerts merges two alerts.
func mergeManyAlerts(previousAlerts, presentAlerts []*v1.Alert) (newAlerts, updatedAlerts, staleAlerts []*v1.Alert) {
	// Merge any alerts that have new and old alerts.
	for _, alert := range presentAlerts {
		if matchingOld := findAlert(alert, previousAlerts); matchingOld != nil {
			updatedAlerts = append(updatedAlerts, mergeAlerts(matchingOld, alert))
		} else {
			alert.FirstOccurred = ptypes.TimestampNow()
			newAlerts = append(newAlerts, alert)
		}
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
