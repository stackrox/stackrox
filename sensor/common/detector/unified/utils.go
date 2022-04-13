package unified

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/detection"
	"github.com/stackrox/stackrox/pkg/set"
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
