package datastore

import (
	globaldb "github.com/stackrox/stackrox/central/globaldb"
	globalDackbox "github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/nodecomponentedge/index"
	"github.com/stackrox/stackrox/central/nodecomponentedge/search"
	"github.com/stackrox/stackrox/central/nodecomponentedge/store"
	"github.com/stackrox/stackrox/central/nodecomponentedge/store/dackbox"
	"github.com/stackrox/stackrox/central/nodecomponentedge/store/postgres"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
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

	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(globaldb.GetPostgres())
		indexer = postgres.NewIndexer(globaldb.GetPostgres())
		searcher = search.New(storage, indexer)
	} else {
		graphProvider = globalDackbox.GetGlobalDackBox()
		storage = dackbox.New(globalDackbox.GetGlobalDackBox())
		indexer = index.New(globalindex.GetGlobalIndex())
		searcher = search.New(storage, indexer)
	}

	ad = New(graphProvider, storage, indexer, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
