package policies

import (
	"github.com/stackrox/rox/generated/storage"
)

// AppliesAtBuildTime returns if a policy applies at build time.
func AppliesAtBuildTime(policy *storage.Policy) bool {
	return appliesAt(policy, storage.LifecycleStage_BUILD)
}

// AppliesAtDeployTime returns if a policy applies at deploy time.
func AppliesAtDeployTime(policy *storage.Policy) bool {
	return appliesAt(policy, storage.LifecycleStage_DEPLOY)
}

// AppliesAtRunTime returns if a policy applies at run time.
func AppliesAtRunTime(policy *storage.Policy) bool {
	return appliesAt(policy, storage.LifecycleStage_RUNTIME)
}

func appliesAt(policy *storage.Policy, lc storage.LifecycleStage) bool {
	for _, stage := range policy.GetLifecycleStages() {
		if stage == lc {
			return true
		}
	}
	return false
}
