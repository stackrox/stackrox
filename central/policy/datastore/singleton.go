package datastore

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/search"
	"github.com/stackrox/rox/central/policy/store"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := store.New(globaldb.GetGlobalDB())
	indexer := index.New(globalindex.GetGlobalIndex())

	searcher, err := search.New(storage, indexer)
	if err != nil {
		panic("unable to load search index for alerts")
	}

	ad = New(storage, indexer, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
