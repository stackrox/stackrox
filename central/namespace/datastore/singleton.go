package datastore

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	dackbox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/namespace/index"
	"github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/namespace/store/bolt"
	"github.com/stackrox/rox/central/namespace/store/rocksdb"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	var storage store.Store
	if env.RocksDB.BooleanSetting() {
		storage = rocksdb.New(globaldb.GetRocksDB())
	} else {
		storage = bolt.New(globaldb.GetGlobalDB())
	}
	indexer := index.New(globalindex.GetGlobalTmpIndex())

	var err error
	as, err = New(storage, dackbox.GetGlobalDackBox(), indexer, deploymentDataStore.Singleton(), ranking.NamespaceRanker())
	utils.Must(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
