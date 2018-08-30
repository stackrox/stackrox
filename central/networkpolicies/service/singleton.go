package service

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/networkgraph"
	"github.com/stackrox/rox/central/networkpolicies/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(store.Singleton(), deploymentDataStore.Singleton(), networkgraph.Singleton(), clusterDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
