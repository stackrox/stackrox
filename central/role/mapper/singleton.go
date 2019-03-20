package mapper

import (
	"github.com/stackrox/rox/pkg/sync"

	groupStore "github.com/stackrox/rox/central/group/store"
	roleStore "github.com/stackrox/rox/central/role/store"
	userStore "github.com/stackrox/rox/central/user/store"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

var (
	roleMapperFactory permissions.RoleMapperFactory
	once              sync.Once
)

// FactorySingleton returns the singleton user role mapper factory.
func FactorySingleton() permissions.RoleMapperFactory {
	once.Do(func() {
		roleMapperFactory = NewStoreBasedMapperFactory(groupStore.Singleton(), roleStore.Singleton(), userStore.Singleton())
	})
	return roleMapperFactory
}
