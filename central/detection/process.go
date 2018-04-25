package detection

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
)

// ProcessDeploymentEvent takes in a deployment event and return alerts.
func (d *Detector) ProcessDeploymentEvent(deployment *v1.Deployment, action v1.ResourceAction) (alertID string, enforcement v1.EnforcementAction) {
	if _, err := d.enricher.Enrich(deployment); err != nil {
		logger.Errorf("Error enriching deployment %s: %s", deployment.GetName(), err)
	}

	var enforcementActions []alertWithEnforcement

	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	for _, policy := range d.policies {
		alert, enforceAction := d.processTask(Task{deployment, action, policy})

		if enforceAction != v1.EnforcementAction_UNSET_ENFORCEMENT {
			enforcementActions = append(enforcementActions, alertWithEnforcement{
				alert:       alert,
				enforcement: enforceAction,
			})
		}
	}

	alertID, enforcement = d.determineEnforcementResponse(enforcementActions)
	return
}

func (d *Detector) processTask(task Task) (alert *v1.Alert, enforcement v1.EnforcementAction) {
	d.markExistingAlertsAsStale(task.deployment.GetId(), task.policy.GetId())

	// No further processing is needed when a deployment is removed.
	if task.action == v1.ResourceAction_REMOVE_RESOURCE {
		return
	}

	deploy, exists, err := d.database.GetDeployment(task.deployment.GetId())
	if err != nil {
		logger.Error(err)
	} else if !exists || deploy.Version != task.deployment.Version {
		return
	}

	// The third argument is if the task matched a whitelist
	var excluded *v1.DryRunResponse_Excluded
	alert, enforcement, excluded = d.Detect(task)

	if alert != nil {
		logger.Warnf("Alert Generated: %v with Severity %v due to policy %v", alert.Id, alert.GetPolicy().GetSeverity().String(), alert.GetPolicy().GetName())
		for _, violation := range alert.GetViolations() {
			logger.Warnf("\t %v", violation.Message)
		}
		if err := d.database.AddAlert(alert); err != nil {
			logger.Error(err)
		}
		d.notificationProcessor.ProcessAlert(alert)
	} else if excluded != nil {
		logger.Infof("Alert for policy '%v' on deployment '%v' was NOT generated due to whitelist '%v'", task.policy.GetName(), task.deployment.GetName(), excluded.GetWhitelist().GetName())
	}

	// This is the best place to assess risk (which is relatively cheap at the moment), because enrichment must have occurred at this point
	// Any new violations (which will soon be integrated into the risk score) will also trigger the reprocessing
	if err := d.enricher.ReprocessDeploymentRisk(task.deployment); err != nil {
		logger.Errorf("Error enriching deployment %s: %s", task.deployment.GetName(), err)
	}

	return
}

func (d *Detector) markExistingAlertsAsStale(deploymentID, policyID string) {
	existingAlerts := d.getExistingAlert(deploymentID, policyID)

	for _, a := range existingAlerts {
		a.Stale = true
		a.MarkedStale = ptypes.TimestampNow()
		if err := d.database.UpdateAlert(a); err != nil {
			logger.Errorf("unable to update alert staleness: %s", err)
		}
	}
}

func (d *Detector) getExistingAlert(deploymentID, policyID string) (existingAlerts []*v1.Alert) {
	var err error
	existingAlerts, err = d.database.GetAlerts(&v1.GetAlertsRequest{
		Stale:        []bool{false},
		DeploymentId: deploymentID,
		PolicyId:     policyID,
	})
	if err != nil {
		logger.Errorf("unable to get alert: %s", err)
		return
	}

	return
}

// Each alert can have an enforcement response, but (assuming that enforcement is mutually exclusive) only one can be
// taken per deployment.
// Scale to Zero Replicas takes precedence over unsatisfiable node constraints.
func (d *Detector) determineEnforcementResponse(enforcementActions []alertWithEnforcement) (alertID string, action v1.EnforcementAction) {
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

type alertWithEnforcement struct {
	alert       *v1.Alert
	enforcement v1.EnforcementAction
}
