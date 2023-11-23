package gcp

import (
	"github.com/stackrox/rox/pkg/cloudproviders/gcp"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	manager gcp.STSClientManager

	log = logging.LoggerForModule()
)

// Singleton returns an instance of the GCP cloud credentials manager.
func Singleton() gcp.STSClientManager {
	once.Do(func() {
		manager = gcp.NewSTSClientManager(
			env.Namespace.Setting(),
			env.GCPCloudCredentialsSecret.Setting(),
		)
	})
	return manager
}
