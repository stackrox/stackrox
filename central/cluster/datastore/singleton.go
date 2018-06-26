package datastore

import (
	"sync"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	"bitbucket.org/stack-rox/apollo/central/cluster/store"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	dnrStore "bitbucket.org/stack-rox/apollo/central/dnrintegration/store"
	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
)

var (
	once sync.Once

	storage store.Store

	ad DataStore
)

func initialize() {
	storage = store.New(globaldb.GetGlobalDB())

	ad = New(storage, alertDataStore.Singleton(), deploymentDataStore.Singleton(), dnrStore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
