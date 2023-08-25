package utils

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// Converts a slice of ResourceWithAccess to a slice of *v1.Permission.
func resourcesWithAccessToPermissions(resourceWithAccess ...permissions.ResourceWithAccess) []*v1.Permission {
	var perms []*v1.Permission
	for _, rAndA := range resourceWithAccess {
		perms = append(perms, &v1.Permission{
			Resource: string(rAndA.Resource.GetResource()),
			Access:   rAndA.Access,
		})
	}
	return perms
}

// FromProtos combines permissions into a map by resource name, using the
// maximum access level for any resource with more than one permission set.
func FromProtos(permissions ...*v1.Permission) map[string]storage.Access {
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

// FromResourcesWithAccess converts a slice of ResourceWithAccess to
// map[string]storage.Access.
func FromResourcesWithAccess(resourceWithAccess ...permissions.ResourceWithAccess) map[string]storage.Access {
	return FromProtos(resourcesWithAccessToPermissions(resourceWithAccess...)...)
}

// NewUnionPermissions returns maximum of the permissions of all input roles.
func NewUnionPermissions(roles []permissions.ResolvedRole) map[string]storage.Access {
	if len(roles) == 0 {
		return nil
	}
	if len(roles) == 1 {
		return roles[0].GetPermissions()
	}

	// Combine permissions into a map by resource, using the maximum access
	// level for any resource with more than one permission set.
	result := make(map[string]storage.Access)
	for _, role := range roles {
		for resource, access := range role.GetPermissions() {
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

	return result
}

func maxAccess(access1, access2 storage.Access) storage.Access {
	if access1 > access2 {
		return access1
	}
	return access2
}

// RoleNames returns a string slice with the names of all given roles.
func RoleNames(roles []permissions.ResolvedRole) []string {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.GetRoleName())
	}
	return names
}

// RoleNamesFromUserInfo converts each UserInfo_Role to role name.
func RoleNamesFromUserInfo(roles []*storage.UserInfo_Role) []string {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.GetName())
	}
	return names
}

// ExtractRolesForUserInfo converts each ResolvedRole to *storage.Role.
func ExtractRolesForUserInfo(roles []permissions.ResolvedRole) []*storage.UserInfo_Role {
	result := make([]*storage.UserInfo_Role, 0, len(roles))
	for _, role := range roles {
		role := &storage.UserInfo_Role{
			Name:             role.GetRoleName(),
			ResourceToAccess: role.GetPermissions(),
		}
		result = append(result, role)
	}
	return result
}
