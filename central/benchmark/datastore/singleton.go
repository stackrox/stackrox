package datastore

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/benchmark/store"
	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
)

var (
	once sync.Once

	storage store.Store

	ad DataStore
)

func initialize() {
	storage = store.New(globaldb.GetGlobalDB())

	var err error
	ad, err = New(storage)
	if err != nil {
		panic(err)
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
