package mapper

import (
	groupStore "github.com/stackrox/rox/central/group/store"
	roleStore "github.com/stackrox/rox/central/role/store"
	userStore "github.com/stackrox/rox/central/user/store"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// NewStoreBasedMapperFactory returns a new instance of a Factory which will use the given stores to create RoleMappers.
func NewStoreBasedMapperFactory(groupStore groupStore.Store, roleStore roleStore.Store, userStore userStore.Store) permissions.RoleMapperFactory {
	return &storeBasedMapperFactoryImpl{
		groupStore: groupStore,
		roleStore:  roleStore,
		userStore:  userStore,
	}
}

type storeBasedMapperFactoryImpl struct {
	groupStore groupStore.Store
	roleStore  roleStore.Store
	userStore  userStore.Store
}

// GetRoleMapper returns a role mapper for the given auth provider.
func (rm *storeBasedMapperFactoryImpl) GetRoleMapper(authProviderID string) permissions.RoleMapper {
	return &storeBasedMapperImpl{
		authProviderID: authProviderID,
		groupStore:     rm.groupStore,
		roleStore:      rm.roleStore,
		userStore:      rm.userStore,
	}
}
