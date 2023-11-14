package manager

import (
	blobstore "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	instance     Manager
	instanceInit sync.Once
)

// Singleton returns the singleton instance for the probe upload manager.
func Singleton() Manager {
	instanceInit.Do(func() {
		instance = newManager(blobstore.Singleton())
		if err := instance.Initialize(); err != nil {
			utils.Should(err)
			log.Error("There was an error initializing the probe upload functionality. Probe upload/download functionality will likely be affected")
		}
	})
	return instance
}
