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

// GetBindingsForRole returns the set of bindings (clusterrolebindings and rolebindings) that have the given role as a roleref
func GetBindingsForRole(role *storage.K8SRole, bindings []*storage.K8SRoleBinding) []*storage.K8SRoleBinding {
	bindingsForRole := make([]*storage.K8SRoleBinding, 0)
	for _, binding := range bindings {
		if binding.GetRoleId() == role.GetId() {
			bindingsForRole = append(bindingsForRole, binding)
		}
	}
	return bindingsForRole
}
