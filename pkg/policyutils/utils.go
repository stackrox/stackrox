package policyutils

import (
	"sort"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

// FillSortHelperFields fills multi word sort fields such as Name, Lifecycle Stages etc.
func FillSortHelperFields(policies ...*storage.Policy) {
	for _, policy := range policies {
		policy.SORTName = policy.Name

		sort.Slice(policy.GetLifecycleStages(), func(i, j int) bool {
			return policy.GetLifecycleStages()[i].String() < policy.GetLifecycleStages()[j].String()
		})
		var stages []string
		for _, lifecycleStage := range policy.GetLifecycleStages() {
			stages = append(stages, lifecycleStage.String())
		}
		policy.SORTLifecycleStage = strings.Join(stages, ",")

		if len(policy.GetEnforcementActions()) > 0 {
			policy.SORTEnforcement = true
		}
	}
}
