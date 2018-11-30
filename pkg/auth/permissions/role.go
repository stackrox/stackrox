package permissions

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// NewRoleWithGlobalAccess returns a new role with the given name,
// which has access to all resources.
func NewRoleWithGlobalAccess(name string, globalAccessLevel v1.Access) *v1.Role {
	return &v1.Role{
		Name:         name,
		GlobalAccess: globalAccessLevel,
	}
}

// NewRoleWithPermissions returns a new role with the given name and permissions.
func NewRoleWithPermissions(name string, permissions ...*v1.Permission) *v1.Role {
	// Combine permissions into a map by resource, using the maximum access level for any
	// resource with more than one permission set.
	resourcetoAccess := make(map[string]v1.Access, len(permissions))
	for _, permission := range permissions {
		if access, exists := resourcetoAccess[permission.GetResource()]; exists {
			resourcetoAccess[permission.GetResource()] = maxAccess(access, permission.GetAccess())
		} else {
			resourcetoAccess[permission.GetResource()] = permission.GetAccess()
		}
	}

	return &v1.Role{
		Name:             name,
		ResourceToAccess: resourcetoAccess,
	}
}

// NewUnionRole returns a new role with maximum of the permissions of all input roles.
func NewUnionRole(roles []*v1.Role) *v1.Role {
	if len(roles) == 0 {
		return nil
	}
	if len(roles) == 1 {
		return roles[0]
	}

	// Combine permissions into a map by resource, using the maximum access level for any
	// resource with more than one permission set.
	globalAccess := v1.Access_NO_ACCESS
	resourceToAccess := make(map[string]v1.Access)
	for _, role := range roles {
		if role.GetGlobalAccess() > globalAccess {
			globalAccess = role.GetGlobalAccess()
		}
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

	return &v1.Role{
		GlobalAccess:     globalAccess,
		ResourceToAccess: resourceToAccess,
	}
}

// RoleHasPermission is a helper function that returns if the given roles provides the given permission.
func RoleHasPermission(role *v1.Role, permission *v1.Permission) bool {
	if role.GetGlobalAccess() >= permission.GetAccess() {
		return true
	}
	return role.GetResourceToAccess()[permission.GetResource()] >= permission.GetAccess()
}

func maxAccess(access1, access2 v1.Access) v1.Access {
	if access1 > access2 {
		return access1
	}
	return access2
}
