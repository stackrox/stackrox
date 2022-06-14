package backend

import (
	"github.com/stackrox/stackrox/central/clusterinit/backend/certificate"
	"github.com/stackrox/stackrox/central/clusterinit/store/singleton"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	backendInstance     Backend
	initBackendInstance sync.Once
)

// Singleton returns the cluster init backend singleton instance.
func Singleton() Backend {
	initBackendInstance.Do(func() {
		backendInstance = newBackend(singleton.Singleton(), certificate.NewProvider())
	})
	return backendInstance
}
