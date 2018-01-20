package detection

import (
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// ProcessDeploymentEvent takes in a deployment event and return alerts.
func (d *Detector) ProcessDeploymentEvent(deployment *v1.Deployment, action v1.ResourceAction) (enforcement v1.EnforcementAction, err error) {
	if _, err = d.enrich(deployment); err != nil {
		return
	}

	var enforcementActions []v1.EnforcementAction

	d.policyMutex.Lock()
	defer d.policyMutex.Unlock()

	for _, policy := range d.policies {
		_, enforceAction := d.processTask(task{deployment, action, policy})

		if enforceAction != v1.EnforcementAction_UNSET_ENFORCEMENT {
			enforcementActions = append(enforcementActions, enforceAction)
		}
	}

	enforcement = d.determineEnforcementResponse(enforcementActions)
	return
}

func (d *Detector) processTask(task task) (alert *v1.Alert, enforcement v1.EnforcementAction) {
	d.markExistingAlertsAsStale(task.deployment, task.policy)

	alert, enforcement = d.detect(task)

	if alert != nil {
		logger.Warnf("Alert Generated: %v with Severity %v due to policy %v", alert.Id, alert.GetPolicy().GetSeverity().String(), alert.GetPolicy().GetName())
		for _, violation := range alert.GetViolations() {
			logger.Warnf("\t %v", violation.Message)
		}
		if err := d.database.AddAlert(alert); err != nil {
			logger.Error(err)
		}
		d.notificationProcessor.ProcessAlert(alert)
	}

	return
}

func (d *Detector) markExistingAlertsAsStale(deployment *v1.Deployment, policy *matcher.Policy) {
	existingAlerts := d.getExistingAlert(deployment, policy)

	for _, a := range existingAlerts {
		a.Stale = true
		if err := d.database.UpdateAlert(a); err != nil {
			logger.Errorf("unable to update alert staleness: %s", err)
		}
	}
}

func (d *Detector) getExistingAlert(deployment *v1.Deployment, policy *matcher.Policy) (existingAlerts []*v1.Alert) {
	var err error
	existingAlerts, err = d.database.GetAlerts(&v1.GetAlertsRequest{
		DeploymentId: []string{deployment.GetId()},
		PolicyName:   []string{policy.GetName()},
		Stale:        []bool{false},
	})
	if err != nil {
		logger.Errorf("unable to get alert: %s", err)
		return
	}

	return
}

// Each alert can have an enforcement response, but (assuming that enforcement is mutually exclusive) only one can be
// taken per deployment.
// Currently a Scale to 0 enforcement response is issued if any alert raises this action.
func (d *Detector) determineEnforcementResponse(enforcementActions []v1.EnforcementAction) v1.EnforcementAction {
	for _, a := range enforcementActions {
		if a == v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT {
			return v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT
		}
	}

	return v1.EnforcementAction_UNSET_ENFORCEMENT
}
