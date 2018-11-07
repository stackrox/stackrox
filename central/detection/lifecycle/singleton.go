package lifecycle

import (
	"sync"

	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/alertmanager"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/enrichment"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
)

var (
	once    sync.Once
	manager Manager
)

func initialize() {
	manager = NewManager(enrichment.Singleton(), deploytime.SingletonDetector(), runtime.SingletonDetector(),
		deploymentDatastore.Singleton(), processDatastore.Singleton(), alertmanager.Singleton())
}

// SingletonManager returns the manager instance.
func SingletonManager() Manager {
	once.Do(initialize)
	return manager
}
