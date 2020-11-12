package backend

import (
	"github.com/stackrox/rox/central/clusterinit/datastore"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	backendInstance     Backend
	initBackendInstance sync.Once
)

// Singleton returns the bootstraptoken backend singleton instance.
func Singleton() Backend {
	initBackendInstance.Do(func() {
		backendInstance = newBackend(datastore.Singleton())
	})
	return backendInstance
}
