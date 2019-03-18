package service

import (
	"sync"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	nsDataStore "github.com/stackrox/rox/central/namespace/datastore"
	flowStoreSingleton "github.com/stackrox/rox/central/networkflow/store/singleton"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/networkpolicies/store"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	"github.com/stackrox/rox/central/sensor/service/connection"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(store.Singleton(), deploymentDS.Singleton(), graph.Singleton(), nsDataStore.Singleton(), clusterDS.Singleton(), notifierStore.Singleton(), flowStoreSingleton.Singleton(), connection.ManagerSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
