package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/networkgraph/config/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/networkgraph/config/datastore/internal/store/rocksdb"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once     sync.Once
	instance DataStore
)

// Singleton provides the instance of DataStore to use.
func Singleton() DataStore {
	once.Do(func() {
		if features.PostgresDatastore.Enabled() {
			instance = New(postgres.New(globaldb.GetPostgres()))
		} else {
			instance = New(rocksdb.New(globaldb.GetRocksDB()))
		}
	})
	return instance
}
