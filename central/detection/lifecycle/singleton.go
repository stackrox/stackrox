package lifecycle

import (
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/alertmanager"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/enrichment"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	manager Manager
)

func initialize() {
	manager = NewManager(enrichment.Singleton(), deploytime.SingletonDetector(), runtime.SingletonDetector(),
		deploymentDatastore.Singleton(), processDatastore.Singleton(), imageDataStore.Singleton(), alertmanager.Singleton(),
		riskManager.Singleton(), reprocessor.Singleton())
}

// SingletonManager returns the manager instance.
func SingletonManager() Manager {
	once.Do(initialize)
	return manager
}
