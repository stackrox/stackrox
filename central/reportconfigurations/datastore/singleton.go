package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/reportconfigurations/index"
	"github.com/stackrox/stackrox/central/reportconfigurations/search"
	"github.com/stackrox/stackrox/central/reportconfigurations/store"
	"github.com/stackrox/stackrox/central/reportconfigurations/store/postgres"
	reportConfigStore "github.com/stackrox/stackrox/central/reportconfigurations/store/rocksdb"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton creates a singleton for the report configuration datastore and loads the plugin client config
func Singleton() DataStore {
	once.Do(func() {
		var err error
		var storage store.Store
		if features.PostgresDatastore.Enabled() {
			storage = postgres.New(globaldb.GetPostgres())
		} else {
			storage, err = reportConfigStore.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
		}

		indexer := index.New(globalindex.GetGlobalTmpIndex())
		searcher := search.New(storage, indexer)

		ds, err = New(storage, indexer, searcher)
		if err != nil {
			log.Panicf("Failed to initialize report configurations datastore: %s", err)
		}
	})
	return ds
}
