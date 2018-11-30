package mapper

import (
	groupStore "github.com/stackrox/rox/central/group/store"
	roleStore "github.com/stackrox/rox/central/role/store"
	userStore "github.com/stackrox/rox/central/user/store"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// A Factory provides an interface for generating a role mapper for an auth provider.
type Factory interface {
	GetRoleMapper(authProviderID string) permissions.RoleMapper
}

// NewFactory returns a new instance of a Factory which will use the given stores to create RoleMappers.
func NewFactory(groupStore groupStore.Store, roleStore roleStore.Store, userStore userStore.Store) Factory {
	return &factoryImpl{
		groupStore: groupStore,
		roleStore:  roleStore,
		userStore:  userStore,
	}
}

type factoryImpl struct {
	groupStore groupStore.Store
	roleStore  roleStore.Store
	userStore  userStore.Store
}

// GetRoleMapper returns a role mapper for the given auth provider.
func (rm *factoryImpl) GetRoleMapper(authProviderID string) permissions.RoleMapper {
	return &mapperImpl{
		authProviderID: authProviderID,
		groupStore:     rm.groupStore,
		roleStore:      rm.roleStore,
		userStore:      rm.userStore,
	}
}
