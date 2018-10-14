package lifecycle

import (
	"sync"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/detection/utils"
	"github.com/stackrox/rox/central/enrichment"
)

var (
	once    sync.Once
	manager Manager
)

func initialize() {
	manager = NewManager(enrichment.Singleton(), deploytime.SingletonDetector(), runtime.SingletonDetector(),
		datastore.Singleton(), utils.SingletonAlertManager())
}

// SingletonManager returns the manager instance.
func SingletonManager() Manager {
	once.Do(initialize)
	return manager
}
