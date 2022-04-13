package connection

import (
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	managerInstance     Manager
	managerInstanceInit sync.Once
)

// ManagerSingleton returns the singleton instance for the sensor connection manager.
func ManagerSingleton() Manager {
	managerInstanceInit.Do(func() {
		managerInstance = newManager()
	})

	return managerInstance
}
