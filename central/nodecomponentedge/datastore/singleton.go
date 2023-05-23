package datastore

import (
	globaldb "github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/central/nodecomponentedge/search"
	"github.com/stackrox/rox/central/nodecomponentedge/store"
	pgStore "github.com/stackrox/rox/central/nodecomponentedge/store/postgres"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var graphProvider graph.Provider
	var storage store.Store
	var indexer index.Indexer
	var searcher search.Searcher

	storage = pgStore.New(globaldb.GetPostgres())
	indexer = pgStore.NewIndexer(globaldb.GetPostgres())
	searcher = search.New(storage, indexer)

	ad = New(graphProvider, storage, indexer, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
