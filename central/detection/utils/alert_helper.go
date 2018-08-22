package utils

import (
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/uuid"
)

// PolicyDeploymentAndViolationsToAlert constructs an alert.
func PolicyDeploymentAndViolationsToAlert(policy *v1.Policy, deployment *v1.Deployment, violations []*v1.Alert_Violation) *v1.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &v1.Alert{
		Id:         uuid.NewV4().String(),
		Deployment: deployment,
		Policy:     policy,
		Violations: violations,
		Time:       ptypes.TimestampNow(),
	}
	if action, msg := PolicyAndDeploymentToEnforcement(policy, deployment); action != v1.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &v1.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
	}
	return alert
}

// PolicyAndViolationsToAlert constructs an alert.
func PolicyAndViolationsToAlert(policy *v1.Policy, violations []*v1.Alert_Violation) *v1.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &v1.Alert{
		Id:         uuid.NewV4().String(),
		Policy:     policy,
		Violations: violations,
		Time:       ptypes.TimestampNow(),
	}
	return alert
}
