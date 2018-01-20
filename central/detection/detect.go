package detection

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/golang/protobuf/ptypes"
)

func (d *Detector) detect(task task) (alert *v1.Alert, enforcement v1.EnforcementAction) {
	if !task.policy.ShouldProcess(task.deployment) {
		return
	}

	if alert = d.generateAlert(task); alert != nil {
		enforcement = alert.GetEnforcement().GetAction()
	}

	return
}

func (d *Detector) generateAlert(task task) (alert *v1.Alert) {
	violations := task.policy.Match(task.deployment)
	if len(violations) == 0 {
		return
	}

	alert = &v1.Alert{
		Id:         uuid.NewV4().String(),
		Deployment: task.deployment,
		Policy:     task.policy.Policy,
		Violations: violations,
		Time:       ptypes.TimestampNow(),
	}

	if action, msg := task.policy.GetEnforcementAction(task.deployment, task.action); action != v1.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &v1.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
	}

	return
}
