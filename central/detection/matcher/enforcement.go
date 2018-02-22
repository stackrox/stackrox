package matcher

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// GetEnforcementAction returns the appropriate enforcement action for deployment.
func (p *Policy) GetEnforcementAction(deployment *v1.Deployment, action v1.ResourceAction) (enforcement v1.EnforcementAction, message string) {
	return newEnforcement(p.GetEnforcement()).enforcementAction(deployment, action)
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

type enforcement interface {
	enforcementAction(deployment *v1.Deployment, action v1.ResourceAction) (enforcement v1.EnforcementAction, message string)
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
