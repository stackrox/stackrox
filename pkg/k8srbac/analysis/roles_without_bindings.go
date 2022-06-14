package analysis

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/set"
)

// getRolesWithNoBindings returns a list of roles without any bindings that are not default K8s roles.
func getRolesWithoutBindings(roles []*storage.K8SRole, roleBindings []*storage.K8SRoleBinding) []*storage.K8SRole {
	// Collect all the roles referenced in role bindings.
	referencedRoles := set.NewStringSet()
	for _, binding := range roleBindings {
		referencedRoles.Add(binding.GetRoleId())
	}

	var rolesWithoutRef []*storage.K8SRole
	for _, role := range roles {
		if !k8srbac.IsDefaultRole(role) && !referencedRoles.Contains(role.GetId()) {
			rolesWithoutRef = append(rolesWithoutRef, role)
		}
	}
	return rolesWithoutRef
}
