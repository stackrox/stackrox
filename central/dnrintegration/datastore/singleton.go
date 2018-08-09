package datastore

import (
	"sync"

	"github.com/stackrox/rox/central/dnrintegration/store"
	"github.com/stackrox/rox/central/globaldb"
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
