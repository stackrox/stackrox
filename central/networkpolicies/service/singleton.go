package service

import (
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	nsDataStore "github.com/stackrox/rox/central/namespace/datastore"
	flowStoreSingleton "github.com/stackrox/rox/central/networkflow/store/singleton"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(
		npDS.Singleton(),
		deploymentDS.Singleton(),
		graph.Singleton(),
		nsDataStore.Singleton(),
		clusterDS.Singleton(),
		notifierDS.Singleton(),
		flowStoreSingleton.Singleton(),
		connection.ManagerSingleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
