package permissions

// A Role represents the role granted to a particular entity seeking API authorization.
type Role interface {
	// Name is a string representation of the role.
	Name() string
	// Has represents whether the role has the given permission.
	Has(Permission) bool
}

// RoleStore allows querying roles by name.
type RoleStore interface {
	RoleByName(name string) Role
}

// NewAllAccessRole returns a new role with the given name,
// which has access to all permissions. Use sparingly!
func NewAllAccessRole(name string) Role {
	return &allAccessRoleImpl{name: name}
}

type allAccessRoleImpl struct {
	name string
}

func (a *allAccessRoleImpl) Name() string {
	return a.name
}

func (a *allAccessRoleImpl) Has(permission Permission) bool {
	return true
}

// NewRoleWithPermissions returns a new role with the given name and permissions.
func NewRoleWithPermissions(name string, permissions ...Permission) Role {
	return &permissionedRoleImpl{
		name:        name,
		permissions: NewPermissionSet(permissions...),
	}
}

type permissionedRoleImpl struct {
	name        string
	permissions PermissionSet
}

func (r *permissionedRoleImpl) Name() string {
	return r.name
}

func (r *permissionedRoleImpl) Has(permission Permission) bool {
	return r.permissions.Contains(permission)
}
