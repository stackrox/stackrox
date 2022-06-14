package utils

import (
	"context"

	"github.com/stackrox/stackrox/central/rbac/k8srole/datastore"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/set"
)

func getRolesForBindings(ctx context.Context, roleStore datastore.DataStore, bindings []*storage.K8SRoleBinding) []*storage.K8SRole {
	roleIDs := set.NewStringSet()
	for _, binding := range bindings {
		roleID := binding.GetRoleId()
		if roleID != "" {
			roleIDs.Add(roleID)
		}
	}

	roles := make([]*storage.K8SRole, 0, roleIDs.Cardinality())
	for roleID := range roleIDs {
		role, exists, err := roleStore.GetRole(ctx, roleID)
		if exists && err == nil {
			roles = append(roles, role)
		}
	}
	return roles
}
