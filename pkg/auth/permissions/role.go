package permissions

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// NewRoleWithAccess returns a new role with the given resource accesses.
func NewRoleWithAccess(name string, resourceWithAccess ...ResourceWithAccess) *storage.Role {
	var permissions []*v1.Permission
	for _, rAndA := range resourceWithAccess {
		permissions = append(permissions, &v1.Permission{
			Resource: string(rAndA.Resource.GetResource()),
			Access:   rAndA.Access,
		})
	}
	return NewRoleWithPermissions(name, permissions...)
}

// NewRoleWithPermissions returns a new role with the given name and permissions.
func NewRoleWithPermissions(name string, permissions ...*v1.Permission) *storage.Role {
	// Combine permissions into a map by resource, using the maximum access level for any
	// resource with more than one permission set.
	resourcetoAccess := make(map[string]storage.Access, len(permissions))
	for _, permission := range permissions {
		if access, exists := resourcetoAccess[permission.GetResource()]; exists {
			resourcetoAccess[permission.GetResource()] = maxAccess(access, permission.GetAccess())
		} else {
			resourcetoAccess[permission.GetResource()] = permission.GetAccess()
		}
	}

	return &storage.Role{
		Name:             name,
		ResourceToAccess: resourcetoAccess,
	}
}

// NewUnionRole returns a new role with maximum of the permissions of all input roles.
func NewUnionRole(roles []*storage.Role) *storage.Role {
	if len(roles) == 0 {
		return nil
	}
	if len(roles) == 1 {
		return roles[0]
	}

	// Combine permissions into a map by resource, using the maximum access level for any
	// resource with more than one permission set.
	resourceToAccess := make(map[string]storage.Access)
	for _, role := range roles {
		for resource, access := range role.GetResourceToAccess() {
			if acc, exists := resourceToAccess[resource]; exists {
				resourceToAccess[resource] = maxAccess(acc, access)
			} else {
				resourceToAccess[resource] = access
			}
		}
	}
	if len(resourceToAccess) == 0 {
		resourceToAccess = nil
	}

	return &storage.Role{
		ResourceToAccess: resourceToAccess,
	}
}

// RoleHasPermission is a helper function that returns if the given roles provides the given permission.
func RoleHasPermission(role *storage.Role, perm ResourceWithAccess) bool {
	return role.GetResourceToAccess()[string(perm.Resource.GetResource())] >= perm.Access
}

func maxAccess(access1, access2 storage.Access) storage.Access {
	if access1 > access2 {
		return access1
	}
	return access2
}

// RoleNames returns a string slice with the names of all given roles.
func RoleNames(roles []*storage.Role) []string {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.GetName())
	}
	return names
}

// GetRolesFromStore fetches each of the provided roles from the store. All roles must exist, otherwise an error
// is returned.
func GetRolesFromStore(ctx context.Context, roleStore RoleStore, roleNames []string) ([]*storage.Role, []int, error) {
	roles := make([]*storage.Role, 0, len(roleNames))
	var missingIndices []int
	for i, roleName := range roleNames {
		role, err := roleStore.GetRole(ctx, roleName)
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
