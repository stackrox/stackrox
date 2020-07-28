package datastore

import (
	"github.com/stackrox/rox/central/apitoken/datastore/internal/store"
	"github.com/stackrox/rox/central/apitoken/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/apitoken/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  DataStore
	once sync.Once
)

func initialize() {
	var storage store.Store
	if features.RocksDB.Enabled() {
		storage = rocksdb.New(globaldb.GetRocksDB())
	} else {
		storage = bolt.MustNew(globaldb.GetGlobalDB())
	}
	svc = New(storage)
}

// Singleton returns the API token singleton.
func Singleton() DataStore {
	once.Do(initialize)
	return svc
}
