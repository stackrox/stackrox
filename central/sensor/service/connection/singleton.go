package connection

import (
	"sync"
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
