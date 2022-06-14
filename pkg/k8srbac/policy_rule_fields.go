package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
)

// APIGroupsField is the apiGroups field of a PolicyRule.
func APIGroupsField() PolicyRuleField {
	return NewWildcardable(
		NewPolicyRuleField(
			func(rule *storage.PolicyRule) []string {
				return rule.GetApiGroups()
			},
			func(value []string, rule *storage.PolicyRule) {
				rule.ApiGroups = value
			},
		),
	)
}

// ResourcesField is the resources field of a PolicyRule.
func ResourcesField() PolicyRuleField {
	return NewWildcardable(
		NewPolicyRuleField(
			func(rule *storage.PolicyRule) []string {
				return rule.GetResources()
			},
			func(value []string, rule *storage.PolicyRule) {
				rule.Resources = value
			},
		),
	)
}

// VerbsField is the verbs field of a PolicyRule.
func VerbsField() PolicyRuleField {
	return NewWildcardable(
		NewPolicyRuleField(
			func(rule *storage.PolicyRule) []string {
				return rule.GetVerbs()
			},
			func(value []string, rule *storage.PolicyRule) {
				rule.Verbs = value
			},
		),
	)
}

// NonResourceURLsField is the non resource urls field of a PolicyRule.
func NonResourceURLsField() PolicyRuleField {
	return NewGlobable(
		NewPolicyRuleField(
			func(rule *storage.PolicyRule) []string {
				return rule.GetNonResourceUrls()
			},
			func(value []string, rule *storage.PolicyRule) {
				rule.NonResourceUrls = value
			},
		),
	)
}

// ResourceNamesField is the resource names field of a PolicyRule.
func ResourceNamesField() PolicyRuleField {
	return NewBlankable(
		NewPolicyRuleField(
			func(rule *storage.PolicyRule) []string {
				return rule.GetResourceNames()
			},
			func(value []string, rule *storage.PolicyRule) {
				rule.ResourceNames = value
			},
		),
	)
}

// CoreFields is a helper that returns the fields we usually use.
func CoreFields() []PolicyRuleField {
	return []PolicyRuleField{
		APIGroupsField(),
		ResourcesField(),
		VerbsField(),
		NonResourceURLsField(),
		ResourceNamesField(),
	}
}
