package backend

import (
	"github.com/stackrox/rox/central/clusterinit/store/singleton"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	backendInstance     Backend
	initBackendInstance sync.Once
)

// Singleton returns the cluster init backend singleton instance.
func Singleton() Backend {
	initBackendInstance.Do(func() {
		backendInstance = newBackend(singleton.Singleton())
	})
	return backendInstance
}
