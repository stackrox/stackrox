package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/watchedimage/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/watchedimage/datastore/internal/store/rocksdb"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	instance DataStore
	once     sync.Once
)

// Singleton returns the instance of DataStore to use.
func Singleton() DataStore {
	once.Do(func() {
		if features.PostgresDatastore.Enabled() {
			store := postgres.New(globaldb.GetPostgres())
			instance = New(store)
		} else {
			store, err := rocksdb.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
			instance = New(store)
		}
	})
	return instance
}
