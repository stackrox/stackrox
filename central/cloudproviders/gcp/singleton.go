package gcp

import (
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/auth"
	"github.com/stackrox/rox/pkg/env"
<<<<<<< HEAD
	"github.com/stackrox/rox/pkg/features"
=======
	"github.com/stackrox/rox/pkg/logging"
>>>>>>> 6be89a2369 (address feedback)
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	manager auth.STSClientManager
)

// Singleton returns an instance of the GCP cloud credentials manager.
func Singleton() auth.STSClientManager {
	once.Do(func() {
		manager = auth.NewSTSClientManager(
			env.Namespace.Setting(),
			env.GCPCloudCredentialsSecret.Setting(),
		)
	})
	return manager
}
