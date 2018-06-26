package datastore

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/deployment/index"
	"bitbucket.org/stack-rox/apollo/central/deployment/search"
	"bitbucket.org/stack-rox/apollo/central/deployment/store"
	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
	globalindex "bitbucket.org/stack-rox/apollo/central/globalindex/singletons"
	"bitbucket.org/stack-rox/apollo/central/ranking"
)

var (
	once sync.Once

	indexer  index.Indexer
	storage  store.Store
	searcher search.Searcher

	ad DataStore
)

func initialize() {
	storage = store.New(globaldb.GetGlobalDB(), ranking.NewRanker())
	indexer = index.New(globalindex.GetGlobalIndex())

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
