package datastore

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/dnrintegration/store"
	"bitbucket.org/stack-rox/apollo/central/globaldb"
)

var (
	once sync.Once

	datastore DataStore
)

func initialize() {
	datastore = New(store.New(globaldb.GetGlobalDB()))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return datastore
}
