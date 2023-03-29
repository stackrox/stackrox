package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/policycategoryedge/index"
	"github.com/stackrox/rox/central/policycategoryedge/search"
	"github.com/stackrox/rox/central/policycategoryedge/store"
	pgStore "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	var storage store.Store
	var indexer index.Indexer

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		storage = pgStore.New(globaldb.GetPostgres())
		indexer = pgStore.NewIndexer(globaldb.GetPostgres())
		ds = New(storage, indexer, search.New(storage, indexer))
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
