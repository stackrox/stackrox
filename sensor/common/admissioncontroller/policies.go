package admissioncontroller

import "github.com/stackrox/stackrox/generated/storage"

func isEnforcedDeployTimePolicy(policy *storage.Policy) bool {
	if policy.GetDisabled() {
		return false
	}

	isDeployLifecycle := false
	for _, stage := range policy.GetLifecycleStages() {
		if stage == storage.LifecycleStage_DEPLOY {
			isDeployLifecycle = true
			break
		}
	}
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
