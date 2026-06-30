package connection

import (
	"github.com/stackrox/rox/central/ha/leases"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	managerInstance     Manager
	managerInstanceInit sync.Once

	leaseStoreInstance *leases.Store
)

// SetLeaseStore sets the HA lease store before the manager singleton is created.
func SetLeaseStore(store *leases.Store) {
	leaseStoreInstance = store
}

// ManagerSingleton returns the singleton instance for the sensor connection manager.
func ManagerSingleton() Manager {
	managerInstanceInit.Do(func() {
		managerInstance = NewManager(hashManager.Singleton(), leaseStoreInstance)
	})

	return managerInstance
}
