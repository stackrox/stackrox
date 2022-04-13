package service

import (
	clusterDS "github.com/stackrox/stackrox/central/cluster/datastore"
	deploymentDS "github.com/stackrox/stackrox/central/deployment/datastore"
	nsDataStore "github.com/stackrox/stackrox/central/namespace/datastore"
	networkBaselineDataStore "github.com/stackrox/stackrox/central/networkbaseline/datastore"
	graphConfigDS "github.com/stackrox/stackrox/central/networkgraph/config/datastore"
	networkEntityDS "github.com/stackrox/stackrox/central/networkgraph/entity/datastore"
	"github.com/stackrox/stackrox/central/networkgraph/entity/networktree"
	nfDS "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	npDS "github.com/stackrox/stackrox/central/networkpolicies/datastore"
	"github.com/stackrox/stackrox/central/networkpolicies/graph"
	notifierDS "github.com/stackrox/stackrox/central/notifier/datastore"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(
		npDS.Singleton(),
		deploymentDS.Singleton(),
		networkEntityDS.Singleton(),
		graphConfigDS.Singleton(),
		networkBaselineDataStore.Singleton(),
		networktree.Singleton(),
		graph.Singleton(),
		nsDataStore.Singleton(),
		clusterDS.Singleton(),
		notifierDS.Singleton(),
		nfDS.Singleton(),
		connection.ManagerSingleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
