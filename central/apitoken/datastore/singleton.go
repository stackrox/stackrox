package datastore

import (
	"github.com/stackrox/rox/central/apitoken/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/apitoken/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/apitoken/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  DataStore
	once sync.Once
)

func initialize() {
	var storage store.Store
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		storage = pgStore.New(globaldb.GetPostgres())
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
