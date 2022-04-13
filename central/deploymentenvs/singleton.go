package deploymentenvs

import "github.com/stackrox/stackrox/pkg/sync"

var (
	managerInstance     Manager
	managerInstanceInit sync.Once
)

// ManagerSingleton returns the singleton instance of the deployment environments manager.
func ManagerSingleton() Manager {
	managerInstanceInit.Do(func() {
		managerInstance = newManager()
	})
	return managerInstance
}
