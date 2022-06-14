package datastore

import (
	globaldb "github.com/stackrox/rox/central/globaldb"
	globalDackbox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/central/nodecomponentedge/search"
	"github.com/stackrox/rox/central/nodecomponentedge/store"
	"github.com/stackrox/rox/central/nodecomponentedge/store/dackbox"
	"github.com/stackrox/rox/central/nodecomponentedge/store/postgres"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/features"
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
