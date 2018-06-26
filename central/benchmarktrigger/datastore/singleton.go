package datastore

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/benchmarktrigger/store"
	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
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
