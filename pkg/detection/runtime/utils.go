package runtime

import (
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
	"github.com/stackrox/rox/pkg/uuid"
)

// PolicyDeploymentAndViolationsToAlert constructs an alert.
func policyDeploymentAndViolationsToAlert(policy *storage.Policy, deployment *storage.Deployment, violations searchbasedpolicies.Violations) *storage.Alert {
	if len(violations.AlertViolations) == 0 && violations.ProcessViolation == nil {
		return nil
	}
	alert := &storage.Alert{
		Id:               uuid.NewV4().String(),
		LifecycleStage:   storage.LifecycleStage_RUNTIME,
		Deployment:       convert.ToAlertDeployment(deployment),
		Policy:           policy.Clone(),
		Violations:       violations.AlertViolations,
		ProcessViolation: violations.ProcessViolation,
		Time:             ptypes.TimestampNow(),
	}
	if action, msg := buildEnforcement(policy, deployment); action != storage.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &storage.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
	}
	return alert
}

func buildEnforcement(policy *storage.Policy, deployment *storage.Deployment) (enforcement storage.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == storage.EnforcementAction_KILL_POD_ENFORCEMENT {
			return storage.EnforcementAction_KILL_POD_ENFORCEMENT, fmt.Sprintf("Deployment %s has pods killed in response to policy violation", deployment.GetName())
		}
	}
	return storage.EnforcementAction_UNSET_ENFORCEMENT, ""
}
