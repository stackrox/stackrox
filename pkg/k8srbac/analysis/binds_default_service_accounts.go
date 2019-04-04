package analysis

import (
	"github.com/stackrox/rox/generated/storage"
)

// getRoleBindingsForDefaultServiceAccounts returns a list of the bindings that bind a default service account to a role.
func getRoleBindingsForDefaultServiceAccounts(roleBindings []*storage.K8SRoleBinding) []*storage.K8SRoleBinding {
	var bindingsForDefaultServiceAccounts []*storage.K8SRoleBinding
	for _, binding := range roleBindings {
		if !IsDefaultRoleBinding(binding) && bindsDefaultServiceAccount(binding) {
			bindingsForDefaultServiceAccounts = append(bindingsForDefaultServiceAccounts, binding)
		}
	}
	return bindingsForDefaultServiceAccounts
}

// bindsDefaultServiceAccount returns if the role binding binds a default service account to a role.
func bindsDefaultServiceAccount(roleBinding *storage.K8SRoleBinding) bool {
	for _, subject := range roleBinding.GetSubjects() {
		if IsDefaultServiceAccountSubject(subject) {
			return true
		}
	}
	return false
}
