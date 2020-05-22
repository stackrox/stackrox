package booleanpolicy

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/set"
)

var (
	runtimeFields = set.NewFrozenStringSet(
		fieldnames.ProcessName, fieldnames.ProcessArguments, fieldnames.ProcessAncestor, fieldnames.ProcessUID,
		fieldnames.WhitelistsEnabled,
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
