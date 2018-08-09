package detection

import (
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/uuid"
)

// Detect takes a Task and returns whether an alert and an enforcement action would be taken
func (d *detectorImpl) Detect(task Task) (alert *v1.Alert, enforcement v1.EnforcementAction, excluded *v1.DryRunResponse_Excluded) {
	if !task.policy.ShouldProcess(task.deployment) {
		return
	}

	if alert, excluded = d.generateAlert(task); alert != nil {
		enforcement = alert.GetEnforcement().GetAction()
	}
	return
}

func (d *detectorImpl) generateAlert(task Task) (alert *v1.Alert, excluded *v1.DryRunResponse_Excluded) {
	violations := task.policy.Match(task.deployment)
	if len(violations) == 0 {
		return
	}

	alert = &v1.Alert{
		Id:         uuid.NewV4().String(),
		Deployment: task.deployment,
		Policy:     task.policy.GetProto(),
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
