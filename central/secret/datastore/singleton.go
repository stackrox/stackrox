package datastore

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/central/secret/store"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	store := store.New(globaldb.GetGlobalDB())
	ad = New(store, index.New(globalindex.GetGlobalIndex()), search.New(store, globalindex.GetGlobalIndex()))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
