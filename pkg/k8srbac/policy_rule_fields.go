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
