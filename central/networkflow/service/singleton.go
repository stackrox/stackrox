package service

import (
	"sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	networkFlowStore "github.com/stackrox/rox/central/networkflow/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(networkFlowStore.Singleton(), deploymentDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
