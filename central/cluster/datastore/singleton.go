package datastore

import (
	"sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/store"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	dnrDataStore "github.com/stackrox/rox/central/dnrintegration/datastore"
	"github.com/stackrox/rox/central/globaldb"
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
