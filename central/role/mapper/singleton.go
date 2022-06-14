package mapper

import (
	groupDataStore "github.com/stackrox/stackrox/central/group/datastore"
	roleDataStore "github.com/stackrox/stackrox/central/role/datastore"
	userDataStore "github.com/stackrox/stackrox/central/user/datastore"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	roleMapperFactory permissions.RoleMapperFactory
	once              sync.Once
)

// FactorySingleton returns the singleton user role mapper factory.
func FactorySingleton() permissions.RoleMapperFactory {
	once.Do(func() {
		roleMapperFactory = NewStoreBasedMapperFactory(groupDataStore.Singleton(), roleDataStore.Singleton(), userDataStore.Singleton())
	})
	return roleMapperFactory
}
