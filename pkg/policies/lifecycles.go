package policies

import "github.com/stackrox/rox/generated/api/v1"

// AppliesAtBuildTime returns if a policy applies at build time.
func AppliesAtBuildTime(policy *v1.Policy) bool {
	return appliesAt(policy, v1.LifecycleStage_BUILD_TIME)
}

// AppliesAtDeployTime returns if a policy applies at deploy time.
func AppliesAtDeployTime(policy *v1.Policy) bool {
	return appliesAt(policy, v1.LifecycleStage_DEPLOY_TIME)
}

// AppliesAtRunTime returns if a policy applies at run time.
func AppliesAtRunTime(policy *v1.Policy) bool {
	return appliesAt(policy, v1.LifecycleStage_RUN_TIME)
}

func appliesAt(policy *v1.Policy, lc v1.LifecycleStage) bool {
	for _, stage := range policy.GetLifecycleStages() {
		if stage == lc {
			return true
		}
	}
	return false
}
