package store

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// storeImpl implements store.
// Currently, it is just a simple in-mem map, since the roles are pre-defined in the code.
// We don't need to persist unless we allow users to create/modify roles.
type storeImpl struct {
	// We don't need to synchronize access to the map with a lock right now,
	// since we only ever read from it. If we start modifying it, we will need
	// to do that.
	roles map[string]*v1.Role
}

func (s *storeImpl) GetRoles() []*v1.Role {
	roles := make([]*v1.Role, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}
	return roles
}

func (s *storeImpl) RoleByName(name string) *v1.Role {
	return s.roles[name]
}
