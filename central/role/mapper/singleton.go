package mapper

import (
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	userDataStore "github.com/stackrox/rox/central/user/datastore"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sync"
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
