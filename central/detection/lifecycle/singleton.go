package lifecycle

import (
	"github.com/stackrox/rox/central/deployment/cache"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/alertmanager"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/enrichment"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processindicator/filter"
	whitelistDataStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	manager *managerImpl
)

func initialize() {
	manager = newManager(
		enrichment.Singleton(),
		deploytime.SingletonDetector(),
		runtime.SingletonDetector(),
		deploymentDatastore.Singleton(),
		processDatastore.Singleton(),
		whitelistDataStore.Singleton(),
		imageDataStore.Singleton(),
		alertmanager.Singleton(),
		riskManager.Singleton(),
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
