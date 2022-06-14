package unified

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/set"
)

func isLifecycleStage(policy *storage.Policy, stage storage.LifecycleStage) bool {
	for _, s := range policy.GetLifecycleStages() {
		if s == stage {
			return true
		}
	}
	return false
}

func reconcilePolicySets(newList []*storage.Policy, policySet detection.PolicySet, matcher func(p *storage.Policy) bool) {
	policyIDSet := set.NewStringSet()
	for _, v := range policySet.GetCompiledPolicies() {
		policyIDSet.Add(v.Policy().GetId())
	}

	for _, p := range newList {
		if !matcher(p) {
			continue
		}
		if err := policySet.UpsertPolicy(p); err != nil {
			log.Errorf("error upserting policy %q: %v", p.GetName(), err)
			continue
		}
		policyIDSet.Remove(p.GetId())
	}
	for removedPolicyID := range policyIDSet {
		policySet.RemovePolicy(removedPolicyID)
	}
}
