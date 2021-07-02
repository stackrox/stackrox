package permissions

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Converts a slice of ResourceWithAccess to a slice of *v1.Permission.
func resourcesWithAccessToPermissions(resourceWithAccess ...ResourceWithAccess) []*v1.Permission {
	var permissions []*v1.Permission
	for _, rAndA := range resourceWithAccess {
		permissions = append(permissions, &v1.Permission{
			Resource: string(rAndA.Resource.GetResource()),
			Access:   rAndA.Access,
		})
	}
	return permissions
}

// Combines permissions into a map by resource name, using the maximum access
// level for any resource with more than one permission set.
func permissionsToResourceToAccess(permissions ...*v1.Permission) map[string]storage.Access {
	resourceToAccess := make(map[string]storage.Access, len(permissions))
	for _, permission := range permissions {
		if access, exists := resourceToAccess[permission.GetResource()]; exists {
			resourceToAccess[permission.GetResource()] = maxAccess(access, permission.GetAccess())
		} else {
			resourceToAccess[permission.GetResource()] = permission.GetAccess()
		}
	}
	return resourceToAccess
}

// ResourcesWithAccessToResourceToAccess converts a slice of ResourceWithAccess
// to map[string]storage.Access.
func ResourcesWithAccessToResourceToAccess(resourceWithAccess ...ResourceWithAccess) map[string]storage.Access {
	return permissionsToResourceToAccess(resourcesWithAccessToPermissions(resourceWithAccess...)...)
}

// NewRoleWithAccess returns a new role with the given resource accesses.
func NewRoleWithAccess(name string, resourceWithAccess ...ResourceWithAccess) *storage.Role {
	return &storage.Role{
		Name:             name,
		ResourceToAccess: ResourcesWithAccessToResourceToAccess(resourceWithAccess...),
	}
}

// NewRoleWithPermissions returns a new role with the given name and permissions.
func NewRoleWithPermissions(name string, permissions ...*v1.Permission) *storage.Role {
	return &storage.Role{
		Name:             name,
		ResourceToAccess: permissionsToResourceToAccess(permissions...),
	}
}

// NewUnionPermissions returns maximum of the permissions of all input roles.
func NewUnionPermissions(resolvedRoles []*ResolvedRole) *storage.ResourceToAccess {
	if len(resolvedRoles) == 0 {
		return nil
	}
	if len(resolvedRoles) == 1 {
		return &storage.ResourceToAccess{
			ResourceToAccess: resolvedRoles[0].GetResourceToAccess(),
		}
	}

	// Combine permissions into a map by resource, using the maximum access level for any
	// resource with more than one permission set.
	result := make(map[string]storage.Access)
	for _, resolvedRole := range resolvedRoles {
		for resource, access := range resolvedRole.GetResourceToAccess() {
			if acc, exists := result[resource]; exists {
				result[resource] = maxAccess(acc, access)
			} else {
				result[resource] = access
			}
		}
	}
	if len(result) == 0 {
		result = nil
	}

	return &storage.ResourceToAccess{
		ResourceToAccess: result,
	}
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

// ExtractRoles extracts *storage.Role instances from list of resolved roles.
func ExtractRoles(resolvedRoles []*ResolvedRole) []*storage.Role {
	result := make([]*storage.Role, 0, len(resolvedRoles))
	for _, resolvedRole := range resolvedRoles {
		result = append(result, resolvedRole.Role)
	}
	return result
}
