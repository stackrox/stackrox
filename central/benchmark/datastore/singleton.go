package datastore

import (
	"sync"

	"github.com/stackrox/rox/central/benchmark/store"
	"github.com/stackrox/rox/central/globaldb"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := store.New(globaldb.GetGlobalDB())

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
