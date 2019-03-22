package datastore

import (
	"github.com/stackrox/rox/central/alert/index"
	"github.com/stackrox/rox/central/alert/search"
	"github.com/stackrox/rox/central/alert/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once         sync.Once
	soleInstance DataStore
)

func initialize() {
	storage := store.New(globaldb.GetGlobalDB())
	indexer := index.New(globalindex.GetGlobalIndex())

	searcher, err := search.New(storage, indexer)
	if err != nil {
		panic("unable to load search index for alerts")
	}

	soleInstance = New(storage, indexer, searcher)
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(initialize)
	return soleInstance
}
