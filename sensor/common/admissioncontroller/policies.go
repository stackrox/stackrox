package admissioncontroller

import (
	"slices"

	"github.com/stackrox/rox/generated/storage"
)

func isEnforcedDeployTimePolicy(policy *storage.Policy) bool {
	if policy.GetDisabled() {
		return false
	}

	isDeployLifecycle := slices.Contains(policy.GetLifecycleStages(), storage.LifecycleStage_DEPLOY)
	if !isDeployLifecycle {
		return false
	}

	isDeployEnforcement := false
	for _, action := range policy.GetEnforcementActions() {
		if action == storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT ||
			action == storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT {
			isDeployEnforcement = true
			break
		}
	}

	return isDeployEnforcement
}
