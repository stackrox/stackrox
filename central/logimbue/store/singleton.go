package store

import (
	"testing"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/logimbue/store/bolt"
	pgStore "github.com/stackrox/rox/central/logimbue/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	storeInstance     Store
	storeInstanceInit sync.Once
)

// Singleton returns the singleton instance for the sensor connection manager.
func Singleton() Store {
	storeInstanceInit.Do(func() {
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			storeInstance = pgStore.New(globaldb.GetPostgres())
		} else {
			storeInstance = bolt.NewStore(globaldb.GetGlobalDB())
		}
	})

	return storeInstance
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) Store {
	return pgStore.New(pool)
}
