package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkgraph/config/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkgraph/config/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
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
