package manager

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/processor"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reports/scheduler"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Manager

	log = logging.LoggerForModule()
)

func initialize() {
	instance = &managerImpl{
		scheduler: scheduler.New(
			reportConfigDS.Singleton(),
			notifierDataStore.Singleton(),
			clusterDataStore.Singleton(),
			namespaceDataStore.Singleton(),
			roleDataStore.Singleton(),
			processor.Singleton(),
		),
	}
}

// Singleton provides the instance of Manager to use.
func Singleton() Manager {
	once.Do(initialize)
	return instance
}
