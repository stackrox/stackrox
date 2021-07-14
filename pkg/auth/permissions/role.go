package permissions

import (
	"context"

	"github.com/pkg/errors"
)

// GetResolvedRolesFromStore resolves each of the provided roles.
func GetResolvedRolesFromStore(ctx context.Context, roleStore RoleStore, roleNames []string) ([]ResolvedRole, []int, error) {
	roles := make([]ResolvedRole, 0, len(roleNames))
	var missingIndices []int
	for i, roleName := range roleNames {
		role, err := roleStore.GetAndResolveRole(ctx, roleName)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "loading role %q", roleName)
		}
		if role == nil {
			missingIndices = append(missingIndices, i)
			continue
		}
		roles = append(roles, role)
	}
	return roles, missingIndices, nil
}
