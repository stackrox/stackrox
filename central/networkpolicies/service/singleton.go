package service

import (
	"sync"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/networkgraph"
	"github.com/stackrox/rox/central/networkpolicies/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(store.Singleton(), networkgraph.Singleton(), clusterDS.Singleton(), deploymentDS.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
