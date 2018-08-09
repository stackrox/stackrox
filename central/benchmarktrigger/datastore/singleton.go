package datastore

import (
	"sync"

	"github.com/stackrox/rox/central/benchmarktrigger/store"
	"github.com/stackrox/rox/central/globaldb"
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
