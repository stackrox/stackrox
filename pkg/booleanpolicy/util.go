package booleanpolicy

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// ContainsOneOf returns whether the policy contains at least one group with a field of specified type.
func ContainsOneOf(policy *storage.Policy, fieldType RuntimeFieldType) bool {
	for _, section := range policy.GetPolicySections() {
		if SectionContainsFieldOfType(section, fieldType) {
			return true
		}
	}
	return false
}

// HasDiscreteEventSource returns whether the policy contains only fields that
// match the specified event source
func HasDiscreteEventSource(policy *storage.Policy, eventSource storage.EventSource) bool {
	for _, section := range policy.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			if !FieldMetadataSingleton().IsFromEventSource(group.GetFieldName(), eventSource) {
				return false
			}
		}
	}
	return true
}

func SectionContainsEventSource(section *storage.PolicySection, eventSource storage.EventSource) bool {
	for _, group := range section.GetPolicyGroups() {
		if FieldMetadataSingleton().IsFromEventSource(group.GetFieldName(), eventSource) {
			return true
		}
	}
	return false
}

// ContainsRuntimeFields returns whether the policy contains runtime specific fields.
func ContainsRuntimeFields(policy *storage.Policy) bool {
	return ContainsOneOf(policy, AuditLogEvent) || ContainsOneOf(policy, Process) ||
		ContainsOneOf(policy, KubeEvent) || ContainsOneOf(policy, NetworkFlow) ||
		ContainsOneOf(policy, FileAccess)
}

// ContainsDeployTimeFields returns whether the policy contains deploy-time specific fields.
func ContainsDeployTimeFields(policy *storage.Policy) bool {
	for _, section := range policy.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			if !FieldMetadataSingleton().IsDeploymentEventField(group.GetFieldName()) &&
				!FieldMetadataSingleton().IsAuditLogEventField(group.GetFieldName()) {
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

// ContainsValidRuntimeFieldCategorySections checks that policy sections only contain
// compatible runtime field types (as defined by runtimeFieldTypeGroups).
func ContainsValidRuntimeFieldCategorySections(policy *storage.Policy) bool {
	if len(policy.GetPolicySections()) == 0 {
		return false
	}

	var runtimeFieldTypeMap = map[RuntimeFieldType]RuntimeFieldType{
		Process:     Process,
		FileAccess:  Process, // FileAccess events contain process information
		NetworkFlow: NetworkFlow,
		KubeEvent:   KubeEvent,
	}

	for _, section := range policy.GetPolicySections() {
		groupsSeen := set.NewStringSet()

		for _, group := range section.GetPolicyGroups() {
			fieldName := group.GetFieldName()
			metadata := FieldMetadataSingleton().fieldsToQB[fieldName]
			if metadata == nil {
				continue
			}
			for _, fieldType := range metadata.fieldTypes {
				if fieldTypeGroup, ok := runtimeFieldTypeMap[fieldType]; ok {
					groupsSeen.Add(string(fieldTypeGroup))
				}
			}

		}

		// A section can only contain fields from one compatibility group
		if groupsSeen.Cardinality() > 1 {
			return false
		}
	}
	return true
}

// SectionContainsFieldOfType returns true if the policy section contains at least one field
// of provided field type.
func SectionContainsFieldOfType(section *storage.PolicySection, fieldType RuntimeFieldType) bool {
	for _, group := range section.GetPolicyGroups() {
		if FieldMetadataSingleton().FieldIsOfType(group.GetFieldName(), fieldType) {
			return true
		}
	}
	return false
}

// SectionContainsFieldName returns true if the section contains a policy group with the given field name.
func SectionContainsFieldName(section *storage.PolicySection, fieldName string) bool {
	for _, group := range section.GetPolicyGroups() {
		if group.GetFieldName() == fieldName {
			return true
		}
	}
	return false
}

// FilterPolicySections returns a new policy containing only the policy sections that satisfy the predicate.
func FilterPolicySections(policy *storage.Policy, pred func(section *storage.PolicySection) bool) *storage.Policy {
	cloned := policy.CloneVT()
	sections := policy.GetPolicySections()
	cloned.PolicySections = nil
	for _, section := range sections {
		if pred(section) {
			cloned.PolicySections = append(cloned.PolicySections, section)
		}
	}
	return cloned
}
