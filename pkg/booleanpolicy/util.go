package booleanpolicy

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/set"
)

// Following fields represents all runtime policy fields.
var (
	processFields = set.NewFrozenStringSet(
		fieldnames.ProcessName,
		fieldnames.ProcessArguments,
		fieldnames.ProcessAncestor,
		fieldnames.ProcessUID,
		fieldnames.UnexpectedProcessExecuted,
	)

	KubeEventsFields = set.NewFrozenStringSet(
		fieldnames.KubeResource,
		fieldnames.KubeAPIVerb,
	)

	NetworkFlowFields = set.NewFrozenStringSet(
		fieldnames.UnexpectedNetworkFlowDetected,
	)

	runtimeFields = set.NewFrozenStringSet(
		fieldnames.ProcessName,
		fieldnames.ProcessArguments,
		fieldnames.ProcessAncestor,
		fieldnames.ProcessUID,
		fieldnames.UnexpectedProcessExecuted,
		fieldnames.KubeResource,
		fieldnames.KubeAPIVerb,
		fieldnames.UnexpectedNetworkFlowDetected,
	)
)

// ContainsOneOf returns whether the policy contains at least one group with a field in fieldSet.
func ContainsOneOf(policy *storage.Policy, fieldSet set.FrozenStringSet) bool {
	for _, section := range policy.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			if fieldSet.Contains(group.GetFieldName()) {
				return true
			}
		}
	}
	return false
}

// ContainsRuntimeFields returns whether the policy contains runtime specific fields.
func ContainsRuntimeFields(policy *storage.Policy) bool {
	return ContainsOneOf(policy, runtimeFields)
}

// ContainsDeployTimeFields returns whether the policy contains deploy-time specific fields.
func ContainsDeployTimeFields(policy *storage.Policy) bool {
	for _, section := range policy.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			if !runtimeFields.Contains(group.GetFieldName()) {
				return true
			}
		}
	}
	return false
}

// ForEachValueWithFieldName calls the given function for each value in any group with the given fieldName.
// If the function returns false, the iteration early exits.
func ForEachValueWithFieldName(policy *storage.Policy, fieldName string, f func(value string) bool) {
	for _, section := range policy.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			if group.GetFieldName() == fieldName {
				for _, val := range group.GetValues() {
					if !f(val.GetValue()) {
						return
					}
				}
			}
		}
	}
}

// GetValuesWithFieldName returns all the values in the policy in groups with the given field name.
func GetValuesWithFieldName(policy *storage.Policy, fieldName string) []string {
	var out []string
	ForEachValueWithFieldName(policy, fieldName, func(value string) bool {
		out = append(out, value)
		return true
	})
	return out
}

// ContainsValueWithFieldName returns whether the policy contains at least one group with the given field name.
func ContainsValueWithFieldName(policy *storage.Policy, fieldName string) bool {
	var contains bool
	ForEachValueWithFieldName(policy, fieldName, func(value string) bool {
		contains = true
		return false
	})
	return contains

}

// ContainsDiscreteRuntimeFieldCategorySections returns false if the policy groups
// contain combination of process and kubernetes events fields.
func ContainsDiscreteRuntimeFieldCategorySections(policy *storage.Policy) bool {
	if len(policy.GetPolicySections()) == 0 {
		return false
	}

	//atLeastOneRunTimeField := false
	for _, section := range policy.GetPolicySections() {
		//if !SectionContainsOneOf(section, runtimeFields) {
		//	continue
		//}
		var numRuntimeSections int
		if SectionContainsOneOf(section, KubeEventsFields) {
			numRuntimeSections++
		}
		if SectionContainsOneOf(section, processFields) {
			numRuntimeSections++
		}
		if SectionContainsOneOf(section, NetworkFlowFields) {
			numRuntimeSections++
		}
		if numRuntimeSections > 1 {
			return false
		}
		//atLeastOneRunTimeField = true
	}
	return true
}

// SectionContainsOneOf returns true if the policy section contains at least one field from provided field set.
func SectionContainsOneOf(section *storage.PolicySection, fieldSet set.FrozenStringSet) bool {
	for _, group := range section.GetPolicyGroups() {
		if fieldSet.Contains(group.GetFieldName()) {
			return true
		}
	}
	return false
}

// FilterPolicySections returns a new policy containing only the policy sections that satisfy the predicate.
func FilterPolicySections(policy *storage.Policy, pred func(section *storage.PolicySection) bool) *storage.Policy {
	cloned := policy.Clone()
	sections := policy.GetPolicySections()
	cloned.PolicySections = nil
	for _, section := range sections {
		if pred(section) {
			cloned.PolicySections = append(cloned.PolicySections, section)
		}
	}
	return cloned
}
