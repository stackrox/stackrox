package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/reportconfigurations/index"
	"github.com/stackrox/rox/central/reportconfigurations/search"
	reportConfigStore "github.com/stackrox/rox/central/reportconfigurations/store/rocksdb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton creates a singleton for the report configuration datastore and loads the plugin client config
func Singleton() DataStore {
	once.Do(func() {
		reportConfigStore, err := reportConfigStore.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)

		indexer := index.New(globalindex.GetGlobalTmpIndex())
		searcher := search.New(reportConfigStore, indexer)

		ds, err = New(reportConfigStore, indexer, searcher)
		if err != nil {
			log.Panicf("Failed to initialize report configurations datastore: %s", err)
		}
	})
	return ds
}
