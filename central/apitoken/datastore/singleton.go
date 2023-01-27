package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  DataStore
	once sync.Once
)

func initialize() {
	// var storage store.Store
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		// storage = postgres.New(globaldb.GetPostgres())
		svc = NewPostgres(globaldb.GetPostgres())
	} else {
		// storage = rocksdb.New(globaldb.GetRocksDB())
		svc = NewRocks(globaldb.GetRocksDB())
	}
	// svc = New(storage)
}

// Singleton returns the API token singleton.
func Singleton() DataStore {
	once.Do(initialize)
	return svc
}
