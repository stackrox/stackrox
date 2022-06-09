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
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var storage store.Store
	var searcher search.Searcher

	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(globaldb.GetPostgres())
		searcher = search.New(storage, index.New(globalindex.GetGlobalIndex()))
	} else {
		storage = dackbox.New(globalDackbox.GetGlobalDackBox())
		searcher = search.New(storage, index.New(globalindex.GetGlobalIndex()))
	}

	ad = New(globalDackbox.GetGlobalDackBox(), storage, index.New(globalindex.GetGlobalIndex()), searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
