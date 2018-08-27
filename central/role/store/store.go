package store

import (
	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// Store is the store for roles.
// For now, roles are read-only, and valid roles are pre-defined in the code.
// The store's interface can be expanded if we start allowing for the creation
// of custom roles, or for the updating of existing roles.
type Store interface {
	GetRole(name string) (role permissions.Role, exists bool)
	GetRoles() []permissions.Role
}

// defaultRolesMap returns the pre-defined roles that we allow.
func defaultRolesMap() map[string]permissions.Role {
	return map[string]permissions.Role{
		role.Admin: permissions.NewAllAccessRole(role.Admin),
		role.ContinuousIntegration: permissions.NewRoleWithPermissions(role.ContinuousIntegration,
			permissions.View(resources.Detection),
		),
		role.SensorCreator: permissions.NewRoleWithPermissions(role.SensorCreator,
			permissions.View(resources.Cluster),
			permissions.Modify(resources.Cluster),
			permissions.View(resources.ServiceIdentity),
		),
	}
}

// New returns a new store.
func New() Store {
	return &storeImpl{
		roles: defaultRolesMap(),
	}
}
