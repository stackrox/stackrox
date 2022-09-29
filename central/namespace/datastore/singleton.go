package datastore

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	globalDackbox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/idmap"
	"github.com/stackrox/rox/central/namespace/index"
	"github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/namespace/store/postgres"
	"github.com/stackrox/rox/central/namespace/store/rocksdb"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/pkg/dackbox"
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
	var indexer index.Indexer
	var dackBox *dackbox.DackBox
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		storage = postgres.New(globaldb.GetPostgres())
		indexer = postgres.NewIndexer(globaldb.GetPostgres())
	} else {
		dackBox = globalDackbox.GetGlobalDackBox()
		storage = rocksdb.New(globaldb.GetRocksDB())
		indexer = index.New(globalindex.GetGlobalTmpIndex())
	}

	var err error
	as, err = New(storage, dackBox, indexer, deploymentDataStore.Singleton(), ranking.NamespaceRanker(), idmap.StorageSingleton())
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
