package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/postgres"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/rocksdb"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	var undoDeploymentStorage undodeploymentstore.UndoDeploymentStore
	if features.PostgresDatastore.Enabled() {
		undoDeploymentStorage = postgres.New(globaldb.GetPostgres())
	} else {
		var err error
		undoDeploymentStorage, err = rocksdb.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)
	}

	as = New(store.Singleton(), undostore.Singleton(), undoDeploymentStorage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
