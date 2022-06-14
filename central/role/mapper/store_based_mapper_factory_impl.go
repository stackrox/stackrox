package mapper

import (
	groupDataStore "github.com/stackrox/stackrox/central/group/datastore"
	roleDataStore "github.com/stackrox/stackrox/central/role/datastore"
	userDataStore "github.com/stackrox/stackrox/central/user/datastore"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
)

// NewStoreBasedMapperFactory returns a new instance of a Factory which will use the given stores to create RoleMappers.
func NewStoreBasedMapperFactory(groups groupDataStore.DataStore, roles roleDataStore.DataStore, users userDataStore.DataStore) permissions.RoleMapperFactory {
	return &storeBasedMapperFactoryImpl{
		groups: groups,
		roles:  roles,
		users:  users,
	}
}

type storeBasedMapperFactoryImpl struct {
	groups groupDataStore.DataStore
	roles  roleDataStore.DataStore
	users  userDataStore.DataStore
}

// GetRoleMapper returns a role mapper for the given auth provider.
func (rm *storeBasedMapperFactoryImpl) GetRoleMapper(authProviderID string) permissions.RoleMapper {
	return &storeBasedMapperImpl{
		authProviderID: authProviderID,
		groups:         rm.groups,
		roles:          rm.roles,
		users:          rm.users,
	}
}
