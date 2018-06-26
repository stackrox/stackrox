package datastore

import (
	"sync"

	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
	"bitbucket.org/stack-rox/apollo/central/imageintegration/store"
)

var (
	once sync.Once

	storage store.Store

	ad DataStore
)

func initialize() {
	storage = store.New(globaldb.GetGlobalDB())

	ad = New(storage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
