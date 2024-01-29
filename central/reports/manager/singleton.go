package manager

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/processor"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	"github.com/stackrox/rox/central/reports/scheduler"
	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Manager
)

func initialize() {
	collectionDS, collectionQueryRes := collectionDataStore.Singleton()
	instance = &managerImpl{
		scheduler: scheduler.New(
			reportConfigDS.Singleton(),
			notifierDataStore.Singleton(),
			clusterDataStore.Singleton(),
			namespaceDataStore.Singleton(),
			deploymentDataStore.Singleton(),
			collectionDS,
			roleDataStore.Singleton(),
			collectionQueryRes,
			processor.Singleton(),
		),
	}
}

// Singleton provides the instance of Manager to use.
func Singleton() Manager {
	once.Do(initialize)
	return instance
}
