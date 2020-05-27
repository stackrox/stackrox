package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/risk/datastore/internal/index"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	"github.com/stackrox/rox/central/risk/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/risk/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	var storage store.Store
	var err error
	if env.RocksDB.BooleanSetting() {
		storage = rocksdb.New(globaldb.GetRocksDB())
	} else {
		storage, err = bolt.New(globaldb.GetGlobalDB())
		if err != nil {
			log.Panicf("Failed to initialize risk store: %v", err)
		}
	}

	indexer := index.New(globalindex.GetGlobalTmpIndex())
	ad, err = New(storage, indexer, search.New(storage, indexer))
	if err != nil {
		log.Panicf("Failed to initialize risks datastore: %s", err)
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
