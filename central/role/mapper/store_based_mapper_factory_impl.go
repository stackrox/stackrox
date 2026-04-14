package mapper

import (
	"context"

	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	userDataStore "github.com/stackrox/rox/central/user/datastore"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/openshift"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// NewStoreBasedMapperFactory returns a new instance of a Factory which will use the given stores to create RoleMappers.
func NewStoreBasedMapperFactory(groups groupDataStore.DataStore, roles roleDataStore.DataStore, users userDataStore.DataStore, authProviders authproviders.Store) permissions.RoleMapperFactory {
	return &storeBasedMapperFactoryImpl{
		groups:        groups,
		roles:         roles,
		users:         users,
		authProviders: authProviders,
	}
}

type storeBasedMapperFactoryImpl struct {
	groups        groupDataStore.DataStore
	roles         roleDataStore.DataStore
	users         userDataStore.DataStore
	authProviders authproviders.Store
}

// GetRoleMapper returns a role mapper for the given auth provider.
func (rm *storeBasedMapperFactoryImpl) GetRoleMapper(ctx context.Context, authProviderID string) permissions.RoleMapper {
	storeBasedMapper := &storeBasedMapperImpl{
		authProviderID: authProviderID,
		groups:         rm.groups,
		roles:          rm.roles,
		users:          rm.users,
	}
	provider, found, err := rm.authProviders.GetAuthProvider(ctx, authProviderID)
	if err != nil || !found {
		return storeBasedMapper
	}
	if provider.GetType() == openshift.TypeName {
		acmMapper, err := NewACMBasedMapper()
		if err != nil {
			return storeBasedMapper
		}
		return NewComposeMapper(storeBasedMapper, acmMapper)
	}
	return storeBasedMapper
}
