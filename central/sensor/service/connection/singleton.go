package connection

import (
	hashManager "github.com/stackrox/rox/central/hash/manager"
	vmindexratelimiter "github.com/stackrox/rox/central/sensor/service/virtualmachineindex/ratelimiter"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	managerInstance     Manager
	managerInstanceInit sync.Once
)

// ManagerSingleton returns the singleton instance for the sensor connection manager.
func ManagerSingleton() Manager {
	managerInstanceInit.Do(func() {
		managerInstance = NewManager(hashManager.Singleton(), vmindexratelimiter.NewFromEnv())
	})

	return managerInstance
}
