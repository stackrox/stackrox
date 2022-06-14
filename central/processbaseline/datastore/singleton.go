package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/processbaseline/index"
	"github.com/stackrox/stackrox/central/processbaseline/search"
	"github.com/stackrox/stackrox/central/processbaseline/store"
	"github.com/stackrox/stackrox/central/processbaseline/store/postgres"
	"github.com/stackrox/stackrox/central/processbaseline/store/rocksdb"
	"github.com/stackrox/stackrox/central/processbaselineresults/datastore"
	indicatorStore "github.com/stackrox/stackrox/central/processindicator/datastore"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	var storage store.Store
	var indexer index.Indexer
	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(globaldb.GetPostgres())
		indexer = postgres.NewIndexer(globaldb.GetPostgres())
	} else {
		var err error
		storage, err = rocksdb.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)

		indexer = index.New(globalindex.GetGlobalTmpIndex())
	}

	searcher, err := search.New(storage, indexer)
	if err != nil {
		panic("unable to load search index for process baseline")
	}

	ad = New(storage, indexer, searcher, datastore.Singleton(), indicatorStore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
