package analysis

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/k8srbac"
	"github.com/stackrox/stackrox/pkg/set"
)

// getBindingsWithoutRoles returns a list of bindings that reference non-existant roles.
func getBindingsWithoutRoles(roles []*storage.K8SRole, roleBindings []*storage.K8SRoleBinding) []*storage.K8SRoleBinding {
	// Collect all the roles referenced in role bindings.
	existingRoles := set.NewStringSet()
	for _, role := range roles {
		existingRoles.Add(role.GetId())
	}

	var bindingsWithoutRoles []*storage.K8SRoleBinding
	for _, binding := range roleBindings {
		if !k8srbac.IsDefaultRoleBinding(binding) && !existingRoles.Contains(binding.GetRoleId()) {
			bindingsWithoutRoles = append(bindingsWithoutRoles, binding)
		}
	}
	return bindingsWithoutRoles
}
