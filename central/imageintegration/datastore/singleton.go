package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/imageintegration/index"
	"github.com/stackrox/rox/central/imageintegration/search"
	"github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/imageintegration/store/bolt"
	"github.com/stackrox/rox/central/imageintegration/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	dataStore DataStore
)

func initializeDefaultIntegrations(storage store.Store) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	iis, err := storage.GetAll(ctx)
	utils.CrashOnError(err)
	if !env.OfflineModeEnv.BooleanSetting() && len(iis) == 0 {
		// Add default integrations
		for _, ii := range store.DefaultImageIntegrations {
			utils.Must(storage.Upsert(ctx, ii))
		}
	}
}

func initialize() {
	// Create underlying store and datastore.
	var storage store.Store
	var indexer index.Indexer

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		storage = postgres.New(globaldb.GetPostgres())
		indexer = postgres.NewIndexer(globaldb.GetPostgres())
	} else {
		storage = bolt.New(globaldb.GetGlobalDB())
		indexer = index.New(globalindex.GetGlobalTmpIndex())
	}
	initializeDefaultIntegrations(storage)
	searcher := search.New(storage, indexer)
	dataStore = New(storage, indexer, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return dataStore
}
