package manager

import (
	"github.com/stackrox/stackrox/pkg/migrations"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	instance     Manager
	instanceInit sync.Once
)

// Singleton returns the singleton instance for the probe upload manager.
func Singleton() Manager {
	instanceInit.Do(func() {
		instance = newManager(migrations.DBMountPath())
		if err := instance.Initialize(); err != nil {
			utils.Should(err)
			log.Error("There was an error initializing the probe upload functionality. Probe upload/download functionality will likely be affected.")
		}
	})
	return instance
}
