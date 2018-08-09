package datastore

import (
	"sync"

	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/search"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/ranking"
)

var (
	once sync.Once

	indexer  index.Indexer
	storage  store.Store
	searcher search.Searcher

	ad DataStore
)

func initialize() {
	indexer = index.New(globalindex.GetGlobalIndex())
	storage = store.New(globaldb.GetGlobalDB(), ranking.NewRanker())

	var err error
	searcher, err = search.New(storage, indexer)
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
