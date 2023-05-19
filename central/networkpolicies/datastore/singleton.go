package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/index"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore"
	undoPGStore "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/postgres"
	undoRocksDB "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/rocksdb"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	var undoDeploymentStorage undodeploymentstore.UndoDeploymentStore
	var networkPolicyStorage store.Store
	var networkPolicyIndex index.Indexer
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		undoDeploymentStorage = undoPGStore.New(globaldb.GetPostgres())
		networkPolicyStorage = postgres.New(globaldb.GetPostgres())
		networkPolicyIndex = postgres.NewIndexer(globaldb.GetPostgres())
	} else {
		var err error
		undoDeploymentStorage, err = undoRocksDB.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)
		networkPolicyStorage = bolt.New(globaldb.GetGlobalDB())
	}

	as = New(networkPolicyStorage, networkPolicyIndex, undostore.Singleton(), undoDeploymentStorage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
