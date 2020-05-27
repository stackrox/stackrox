package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/index"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/store"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/store/bolt"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/store/rocksdb"
	"github.com/stackrox/rox/central/rbac/k8srole/search"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/storecache"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
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
		storage, err = rocksdb.New(globaldb.GetRocksDB())
	} else {
		storage, err = bolt.NewBoltStore(globaldb.GetGlobalDB(), storecache.NewMapBackedCache())
	}
	utils.Must(err)

	indexer := index.New(globalindex.GetGlobalTmpIndex())
	ad, err = New(storage, indexer, search.New(storage, indexer))
	if err != nil {
		log.Panicf("Failed to initialize k8s role datastore: %s", err)
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
