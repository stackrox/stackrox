package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/networkgraph/config/datastore/internal/store/rocksdb"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once     sync.Once
	instance DataStore
)

// Singleton provides the instance of DataStore to use.
func Singleton() DataStore {
	once.Do(func() {
		instance = New(rocksdb.New(globaldb.GetRocksDB()))
	})
	return instance
}
