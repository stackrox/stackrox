package permissions

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// RoleStore allows querying roles by name.
type RoleStore interface {
	RoleByName(name string) *v1.Role
}

// NewReadWriteRole returns a new role with the given name,
// which has access to all permissions. Use sparingly!
func NewReadWriteRole(name string) *v1.Role {
	return &v1.Role{
		Name:         name,
		GlobalAccess: v1.Access_READ_WRITE_ACCESS,
	}
}

// NewReadOnlyRole returns a new role with the given name that has read only access to all resources.
func NewReadOnlyRole(name string) *v1.Role {
	return &v1.Role{
		Name:         name,
		GlobalAccess: v1.Access_READ_ACCESS,
	}
}

// NewRoleWithPermissions returns a new role with the given name and permissions.
func NewRoleWithPermissions(name string, permissions ...*v1.Permission) *v1.Role {
	// Combine permissions into a map by resource, using the maximum access level for any
	// resource with more than one permission set.
	resourceToPermission := make(map[string]*v1.Permission, len(permissions))
	for _, permission := range permissions {
		if curr, exists := resourceToPermission[permission.GetResource()]; exists {
			resourceToPermission[permission.GetResource()].Access = maxAccess(curr.GetAccess(), permission.GetAccess())
		} else {
			resourceToPermission[permission.GetResource()] = permission
		}
	}

	return &v1.Role{
		Name:                 name,
		ResourceToPermission: resourceToPermission,
	}
}

// RoleHasPermission is a helper function that returns if the given roles provides the given permission.
func RoleHasPermission(role *v1.Role, permission *v1.Permission) bool {
	if role.GetGlobalAccess() >= permission.GetAccess() {
		return true
	}
	return role.GetResourceToPermission()[permission.GetResource()].GetAccess() >= permission.GetAccess()
}

func maxAccess(access1, access2 v1.Access) v1.Access {
	if access1 > access2 {
		return access1
	}
	return access2
}
