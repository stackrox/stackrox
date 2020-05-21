package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/processwhitelistresults/datastore/internal/store"
	"github.com/stackrox/rox/central/processwhitelistresults/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/processwhitelistresults/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	singleton DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	var storage store.Store
	if features.RocksDB.Enabled() {
		storage = rocksdb.New(globaldb.GetRocksDB())
	} else {
		var err error
		storage, err = bolt.NewBoltStore(globaldb.GetGlobalDB())
		utils.Must(err)
	}
	singleton = New(storage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return singleton
}
