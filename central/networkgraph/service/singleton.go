package service

import (
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	graphConfigDataStore "github.com/stackrox/stackrox/central/networkgraph/config/datastore"
	networkEntityDatastore "github.com/stackrox/stackrox/central/networkgraph/entity/datastore"
	"github.com/stackrox/stackrox/central/networkgraph/entity/networktree"
	nfDS "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(nfDS.Singleton(),
		networkEntityDatastore.Singleton(),
		networktree.Singleton(),
		deploymentDataStore.Singleton(),
		clusterDataStore.Singleton(),
		graphConfigDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
