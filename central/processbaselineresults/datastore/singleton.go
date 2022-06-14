package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/processbaselineresults/datastore/internal/store"
	"github.com/stackrox/stackrox/central/processbaselineresults/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/processbaselineresults/datastore/internal/store/rocksdb"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	singleton DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	var storage store.Store
	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(globaldb.GetPostgres())
	} else {
		storage = rocksdb.New(globaldb.GetRocksDB())
	}
	singleton = New(storage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return singleton
}
