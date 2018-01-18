package detection

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func (p *policyWrapper) getEnforcementAction(deployment *v1.Deployment, action v1.ResourceAction) (enforcement v1.EnforcementAction, message string) {
	if !p.GetEnforce() {
		return
	}

	if action != v1.ResourceAction_CREATE_RESOURCE {
		return
	}

	if deployment.GetType() == "Global" || deployment.GetType() == "DaemonSet" {
		return
	}

	return v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, fmt.Sprintf("Deployment %s scaled to 0 replicas in response to policy violation", deployment.GetName())
}
