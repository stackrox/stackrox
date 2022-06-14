package datastore

import (
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/globaldb"
	dackbox "github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/idmap"
	"github.com/stackrox/stackrox/central/namespace/index"
	"github.com/stackrox/stackrox/central/namespace/store"
	"github.com/stackrox/stackrox/central/namespace/store/postgres"
	"github.com/stackrox/stackrox/central/namespace/store/rocksdb"
	"github.com/stackrox/stackrox/central/ranking"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	var storage store.Store
	var indexer index.Indexer
	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(globaldb.GetPostgres())
		indexer = postgres.NewIndexer(globaldb.GetPostgres())
	} else {
		storage = rocksdb.New(globaldb.GetRocksDB())
		indexer = index.New(globalindex.GetGlobalTmpIndex())
	}

	var err error
	as, err = New(storage, dackbox.GetGlobalDackBox(), indexer, deploymentDataStore.Singleton(), ranking.NamespaceRanker(), idmap.StorageSingleton())
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
