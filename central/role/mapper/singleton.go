package mapper

import (
	"sync"

	groupStore "github.com/stackrox/rox/central/group/store"
	roleStore "github.com/stackrox/rox/central/role/store"
	userStore "github.com/stackrox/rox/central/user/store"
)

var (
	roleMapperFactory Factory
	once              sync.Once
)

// FactorySingleton returns the singleton user role mapper factory.
func FactorySingleton() Factory {
	once.Do(func() {
		roleMapperFactory = NewFactory(groupStore.Singleton(), roleStore.Singleton(), userStore.Singleton())
	})
	return roleMapperFactory
}
