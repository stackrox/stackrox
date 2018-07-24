package datastore

import (
	"sync"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	"bitbucket.org/stack-rox/apollo/central/cluster/store"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	dnrDataStore "bitbucket.org/stack-rox/apollo/central/dnrintegration/datastore"
	"bitbucket.org/stack-rox/apollo/central/globaldb"
)

var (
	once sync.Once

	storage store.Store

	ad DataStore
)

func initialize() {
	storage = store.New(globaldb.GetGlobalDB())

	ad = New(storage, alertDataStore.Singleton(), deploymentDataStore.Singleton(), dnrDataStore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
