package compiledpolicies

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

type enforcement interface {
	enforcementAction(deployment *v1.Deployment, action v1.ResourceAction) (enforcement v1.EnforcementAction, message string)
}

func newEnforcement(action v1.EnforcementAction) enforcement {
	switch action {
	case v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT:
		return scaleToZeroEnforcement(action)
	case v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT:
		return nodeConstraintEnforcement(action)
	default:
		return unsetEnforcement(action)
	}
}

type unsetEnforcement v1.EnforcementAction

func (unsetEnforcement) enforcementAction(*v1.Deployment, v1.ResourceAction) (enforcement v1.EnforcementAction, message string) {
	return
}

type scaleToZeroEnforcement v1.EnforcementAction

func (scaleToZeroEnforcement) enforcementAction(deployment *v1.Deployment, action v1.ResourceAction) (enforcement v1.EnforcementAction, message string) {
	if action != v1.ResourceAction_CREATE_RESOURCE {
		return
	}

	if deployment.GetType() == "Global" || deployment.GetType() == "DaemonSet" {
		return
	}

	return v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, fmt.Sprintf("Deployment %s scaled to 0 replicas in response to policy violation", deployment.GetName())
}

type nodeConstraintEnforcement v1.EnforcementAction

func (nodeConstraintEnforcement) enforcementAction(deployment *v1.Deployment, action v1.ResourceAction) (enforcement v1.EnforcementAction, message string) {
	if action != v1.ResourceAction_CREATE_RESOURCE {
		return
	}

	return v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT, fmt.Sprintf("Unsatisfiable node constraint applied to deployment %s", deployment.GetName())
}
