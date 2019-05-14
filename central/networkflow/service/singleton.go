package service

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(nfDS.Singleton(), deploymentDataStore.Singleton(), graph.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
