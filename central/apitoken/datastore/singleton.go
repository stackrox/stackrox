package datastore

import (
	"github.com/stackrox/stackrox/central/apitoken/datastore/internal/store"
	"github.com/stackrox/stackrox/central/apitoken/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/apitoken/datastore/internal/store/rocksdb"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	svc  DataStore
	once sync.Once
)

func initialize() {
	var storage store.Store
	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(globaldb.GetPostgres())
	} else {
		storage = rocksdb.New(globaldb.GetRocksDB())
	}
	svc = New(storage)
}

// Singleton returns the API token singleton.
func Singleton() DataStore {
	once.Do(initialize)
	return svc
}
