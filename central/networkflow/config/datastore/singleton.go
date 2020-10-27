package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkflow/config/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance DataStore
)

// Singleton provides the instance of DataStore to use.
func Singleton() DataStore {
	if !features.NetworkGraphExternalSrcs.Enabled() {
		return nil
	}

	once.Do(func() {
		instance = New(rocksdb.New(globaldb.GetRocksDB()))
	})
	return instance
}
