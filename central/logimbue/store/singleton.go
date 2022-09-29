package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/logimbue/store/bolt"
	"github.com/stackrox/rox/central/logimbue/store/postgres"
	"github.com/stackrox/rox/pkg/env"
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
			storeInstance = postgres.New(globaldb.GetPostgres())
		} else {
			storeInstance = bolt.NewStore(globaldb.GetGlobalDB())
		}
	})

	return storeInstance
}
