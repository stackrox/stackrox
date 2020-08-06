package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/serviceaccount/internal/index"
	"github.com/stackrox/rox/central/serviceaccount/internal/store/rocksdb"
	"github.com/stackrox/rox/central/serviceaccount/search"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	storage, err := rocksdb.New(globaldb.GetRocksDB())
	utils.Must(err)

	indexer := index.New(globalindex.GetGlobalTmpIndex())
	ds, err = New(storage, indexer, search.New(storage, indexer))
	if err != nil {
		log.Panicf("Failed to initialize secrets datastore: %s", err)
	}
}

// Singleton returns a singleton instance of the service account datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
