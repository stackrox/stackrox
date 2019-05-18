package singleton

import (
	"github.com/stackrox/rox/central/deploymentenvs"
	"github.com/stackrox/rox/central/license/datastore"
	"github.com/stackrox/rox/central/license/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	instance     manager.LicenseManager
	instanceInit sync.Once
)

// ManagerSingleton returns the license manager singleton instance
func ManagerSingleton() manager.LicenseManager {
	instanceInit.Do(func() {
		instance = manager.New(datastore.Singleton(), validatorInstance, deploymentenvs.ManagerSingleton())
	})
	return instance
}
