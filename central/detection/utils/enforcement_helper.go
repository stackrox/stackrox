package utils

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

const (
	globalDeployment    = "Global"
	gaemonSetDeployment = "DaemonSet"
)

// DetermineEnforcement returns the alert and its enforcement action to use from the input list (if any have enforcement).
func DetermineEnforcement(alerts []*v1.Alert) (alertID string, action v1.EnforcementAction) {
	for _, alert := range alerts {
		if alert.GetEnforcement() != nil && alert.GetEnforcement().GetAction() == v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT {
			return alert.GetId(), v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT
		}

		if alert.GetEnforcement() != nil && alert.GetEnforcement().GetAction() != v1.EnforcementAction_UNSET_ENFORCEMENT {
			alertID = alert.GetId()
			action = alert.GetEnforcement().GetAction()
		}
	}
	return
}

// PolicyAndDeploymentToEnforcement returns enforcement info for a deployment violating a policy.
func PolicyAndDeploymentToEnforcement(policy *v1.Policy, deployment *v1.Deployment) (enforcement v1.EnforcementAction, message string) {
	if policy.GetEnforcement() == v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT && scaleToZeroEnabled(deployment) {
		return v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, fmt.Sprintf("Deployment %s scaled to 0 replicas in response to policy violation", deployment.GetName())
	}
	if policy.GetEnforcement() == v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT {
		return v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT, fmt.Sprintf("Unsatisfiable node constraint applied to deployment %s", deployment.GetName())
	}
	return v1.EnforcementAction_UNSET_ENFORCEMENT, ""
}

func scaleToZeroEnabled(deployment *v1.Deployment) bool {
	if deployment.GetType() == globalDeployment || deployment.GetType() == gaemonSetDeployment {
		return false
	}
	return true
}
