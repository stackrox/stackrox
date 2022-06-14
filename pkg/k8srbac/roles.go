package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// GetVerbsForRole returns the set of verbs granted by a role.
func GetVerbsForRole(role *storage.K8SRole) set.StringSet {
	policyRuleSet := NewPolicyRuleSet(CoreFields()...)
	policyRuleSet.Add(role.GetRules()...)

	return policyRuleSet.VerbSet()
}

// GetResourcesForRole returns the set of resources that have been granted permissions to by a role.
func GetResourcesForRole(role *storage.K8SRole) set.StringSet {
	policyRuleSet := NewPolicyRuleSet(CoreFields()...)
	policyRuleSet.Add(role.GetRules()...)

	return policyRuleSet.ResourceSet()
}

// GetNonResourceURLsForRole returns the set of non resources urls that have been granted permissions to by a role.
func GetNonResourceURLsForRole(role *storage.K8SRole) set.StringSet {
	policyRuleSet := NewPolicyRuleSet(CoreFields()...)
	policyRuleSet.Add(role.GetRules()...)

	return policyRuleSet.NonResourceURLSet()
}
