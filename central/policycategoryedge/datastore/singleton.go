package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/policycategoryedge/index"
	"github.com/stackrox/rox/central/policycategoryedge/search"
	"github.com/stackrox/rox/central/policycategoryedge/store"
	"github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	var err error
	var storage store.Store
	var indexer index.Indexer

	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(globaldb.GetPostgres())
		indexer = postgres.NewIndexer(globaldb.GetPostgres())
		ds, err = New(storage, indexer, search.NewV2(storage, indexer))
		utils.CrashOnError(err)
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
