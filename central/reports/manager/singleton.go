package manager

import (
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	namespaceDataStore "github.com/stackrox/stackrox/central/namespace/datastore"
	notifierDataStore "github.com/stackrox/stackrox/central/notifier/datastore"
	"github.com/stackrox/stackrox/central/notifier/processor"
	reportConfigDS "github.com/stackrox/stackrox/central/reportconfigurations/datastore"
	"github.com/stackrox/stackrox/central/reports/scheduler"
	roleDataStore "github.com/stackrox/stackrox/central/role/datastore"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sync"
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
