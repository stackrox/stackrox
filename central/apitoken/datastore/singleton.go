package datastore

import (
	"github.com/stackrox/rox/central/apitoken/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  DataStore
	once sync.Once
)

func initialize() {
	storage := rocksdb.New(globaldb.GetRocksDB())
	svc = New(storage)
}

// Singleton returns the API token singleton.
func Singleton() DataStore {
	once.Do(initialize)
	return svc
}
