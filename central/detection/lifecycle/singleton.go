package lifecycle

import (
	"github.com/stackrox/rox/central/deployment/cache"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/alertmanager"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/runtime"
	baselineDataStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processindicator/filter"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	manager *managerImpl
)

func initialize() {
	manager = newManager(
		deploytime.SingletonDetector(),
		runtime.SingletonDetector(),
		deploymentDatastore.Singleton(),
		processDatastore.Singleton(),
		baselineDataStore.Singleton(),
		alertmanager.Singleton(),
		reprocessor.Singleton(),
		cache.DeletedDeploymentCacheSingleton(),
		filter.Singleton(),
	)
	go manager.buildIndicatorFilter()
}

// SingletonManager returns the manager instance.
func SingletonManager() Manager {
	once.Do(initialize)
	return manager
}
