package deploytime

import (
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/uuid"
)

// Label key used for unsatisfiable node constraint enforcement.
const (
	UnsatisfiableNodeConstraintKey = `BlockedByStackRox`
)

// PolicyDeploymentAndViolationsToAlert constructs an alert.
func PolicyDeploymentAndViolationsToAlert(policy *storage.Policy, deployment *storage.Deployment, violations []*storage.Alert_Violation) *storage.Alert {
	if len(violations) == 0 {
		return nil
	}

	alert := &storage.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: storage.LifecycleStage_DEPLOY,
		Entity:         convert.ToAlertDeployment(deployment),
		Policy:         policy.Clone(),
		Violations:     violations,
		Time:           ptypes.TimestampNow(),
	}
	if action, msg := buildEnforcement(policy, deployment); action != storage.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &storage.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
	}
	return alert
}

// buildEnforcement returns the enforcement action and message for deploy time enforcement.
func buildEnforcement(policy *storage.Policy, deployment *storage.Deployment) (enforcement storage.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT && scaleToZeroEnabled(deployment) {
			return storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				"Deployment scaled to 0 replicas in response to this policy violation."
		}
		if enforcementAction == storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT {
			return storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
				fmt.Sprintf("Unsatisfiable node constraint %s applied to deployment %s.", UnsatisfiableNodeConstraintKey, deployment.GetName())
		}
	}
	return storage.EnforcementAction_UNSET_ENFORCEMENT, ""
}

const (
	globalDeployment    = "Global"
	daemonSetDeployment = "DaemonSet"
)

func scaleToZeroEnabled(deployment *storage.Deployment) bool {
	if deployment.GetType() == globalDeployment || deployment.GetType() == daemonSetDeployment {
		return false
	}
	return true
}
