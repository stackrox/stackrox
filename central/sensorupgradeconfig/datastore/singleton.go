package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store/bolt"
	pgStore "github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton DataStore
)

func initialize() {
	var storage store.Store
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		storage = pgStore.New(globaldb.GetPostgres())
	} else {
		storage = bolt.New(globaldb.GetGlobalDB())
	}
	singleton = New(storage)
}

// Singleton returns the datastore instance.
func Singleton() DataStore {
	once.Do(initialize)
	return singleton
}
