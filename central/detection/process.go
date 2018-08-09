package detection

import (
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

type alertWithEnforcement struct {
	alert       *v1.Alert
	enforcement v1.EnforcementAction
}

// ProcessDeploymentEvent takes in a deployment event and return alerts.
func (d *detectorImpl) ProcessDeploymentEvent(deployment *v1.Deployment, action v1.ResourceAction) (alertID string, enforcement v1.EnforcementAction) {
	if _, err := d.enricher.Enrich(deployment); err != nil {
		logger.Errorf("Error enriching deployment %s: %s", deployment.GetName(), err)
	}

	enforcementActions := d.evaluatePolicies(deployment, action)
	alertID, enforcement = d.determineEnforcementResponse(enforcementActions)
	return
}

func (d *detectorImpl) evaluatePolicies(deployment *v1.Deployment, action v1.ResourceAction) []alertWithEnforcement {
	d.policyMutex.RLock()
	defer d.policyMutex.RUnlock()

	var enforcementActions []alertWithEnforcement
	for _, policy := range d.policies {
		alert, enforceAction := d.processTask(Task{deployment, action, policy})

		if enforceAction != v1.EnforcementAction_UNSET_ENFORCEMENT {
			enforcementActions = append(enforcementActions, alertWithEnforcement{
				alert:       alert,
				enforcement: enforceAction,
			})
		}
	}
	return enforcementActions
}

func (d *detectorImpl) reprocessDeploymentRiskAndLogError(deployment *v1.Deployment) {
	if err := d.enricher.ReprocessDeploymentRisk(deployment); err != nil {
		logger.Errorf("Error reprocessing risk for deployment %s: %s", deployment.GetName(), err)
	}
}

func (d *detectorImpl) processTask(task Task) (alert *v1.Alert, enforcement v1.EnforcementAction) {
	existingAlerts := d.getExistingAlerts(task.deployment.GetId(), task.policy.GetProto().GetId())

	// No further processing is needed when a deployment is removed.
	if task.action == v1.ResourceAction_REMOVE_RESOURCE {
		d.markExistingAlertsAsStale(existingAlerts)
		return
	}

	// This is the best place to assess risk (which is relatively cheap at the moment), because enrichment must have occurred at this point
	// Any new violations (which will soon be integrated into the risk score) will also trigger the reprocessing
	defer func() {
		go d.reprocessDeploymentRiskAndLogError(task.deployment)
	}()

	// The third argument is if the task matched a whitelist
	var excluded *v1.DryRunResponse_Excluded
	alert, enforcement, excluded = d.Detect(task)
	// If the task is now whitelisted, whether the policy has whitelisted it or
	// the deployment now falls within the whitelisted scope then remove all alerts
	// and return
	if excluded != nil {
		d.markExistingAlertsAsStale(existingAlerts)
		return
	}

	if alert != nil {
		switch {
		case len(existingAlerts) == 0:
			logger.Debugf("Alert Generated: %s with Severity %s due to policy %s", alert.Id, alert.GetPolicy().GetSeverity().String(), alert.GetPolicy().GetName())
			for _, violation := range alert.GetViolations() {
				logger.Debugf("\t %v", violation.Message)
			}
			alert.FirstOccurred = ptypes.TimestampNow()
			if err := d.alertStorage.AddAlert(alert); err != nil {
				logger.Error(err)
			} else {
				// Don't notify if the save failed. Otherwise, the user can't look up any of the data in the UI
				d.notificationProcessor.ProcessAlert(alert)
			}
		case len(existingAlerts) > 1:
			logger.Errorf("Found more than 1 existing alert for deployment '%s' and policy '%s'", task.deployment.Id, task.policy.GetProto().GetId())
			d.markExistingAlertsAsStale(existingAlerts[1:])
			fallthrough
		case len(existingAlerts) == 1:
			alert = mergeAlerts(existingAlerts[0], alert)
			logger.Debugf("Alert Updated: %s with Severity %s due to policy %s", alert.Id, alert.GetPolicy().GetSeverity().String(), alert.GetPolicy().GetName())
			if err := d.alertStorage.UpdateAlert(alert); err != nil {
				logger.Error(err)
			}
		}
	} else {
		d.markExistingAlertsAsStale(existingAlerts)
	}

	return
}

func (d *detectorImpl) markExistingAlertsAsStale(existingAlerts []*v1.Alert) {
	for _, a := range existingAlerts {
		a.Stale = true
		a.MarkedStale = ptypes.TimestampNow()
		if err := d.alertStorage.UpdateAlert(a); err != nil {
			logger.Errorf("unable to update alert staleness: %s", err)
		}
	}
}

func (d *detectorImpl) getExistingAlerts(deploymentID, policyID string) (existingAlerts []*v1.Alert) {
	qb := search.NewQueryBuilder().
		AddBools(search.Stale, false).
		AddStrings(search.DeploymentID, deploymentID).
		AddStrings(search.PolicyID, policyID)

	var err error
	existingAlerts, err = d.alertStorage.SearchRawAlerts(qb.ToParsedSearchRequest())

	if err != nil {
		logger.Errorf("unable to get alert: %s", err)
		return
	}
	return
}

// Each alert can have an enforcement response, but (assuming that enforcement is mutually exclusive) only one can be
// taken per deployment.
// Scale to Zero Replicas takes precedence over unsatisfiable node constraints.
func (d *detectorImpl) determineEnforcementResponse(enforcementActions []alertWithEnforcement) (alertID string, action v1.EnforcementAction) {
	for _, enfAction := range enforcementActions {
		if enfAction.enforcement == v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT {
			return enfAction.alert.GetId(), v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT
		}

		if enfAction.enforcement != v1.EnforcementAction_UNSET_ENFORCEMENT {
			alertID = enfAction.alert.GetId()
			action = enfAction.enforcement
		}
	}

	return
}

func mergeAlerts(old, new *v1.Alert) *v1.Alert {
	new.Id = old.GetId()
	new.Enforcement = old.GetEnforcement()
	new.FirstOccurred = old.GetFirstOccurred()
	return new
}
