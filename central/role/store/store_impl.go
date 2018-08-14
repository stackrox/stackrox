package store

import "github.com/stackrox/rox/pkg/auth/permissions"

// storeImpl implements store.
// Currently, it is just a simple in-mem map, since the roles are pre-defined in the code.
// We don't need to persist unless we allow users to create/modify roles.
type storeImpl struct {
	// We don't need to synchronize access to the map with a lock right now,
	// since we only ever read from it. If we start modifying it, we will need
	// to do that.
	roles map[string]permissions.Role
}

func (s *storeImpl) GetRoles() []permissions.Role {
	roles := make([]permissions.Role, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}
	return roles
}

func (s *storeImpl) GetRole(name string) (role permissions.Role, exists bool) {
	role, exists = s.roles[name]
	return
}
