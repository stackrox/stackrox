package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once      sync.Once
	singleton DataStore
)

func initialize() {
	var storage store.Store
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		storage = postgres.New(globaldb.GetPostgres())
	} else {
		storage = bolt.New(globaldb.GetGlobalDB())
	}
	var err error
	singleton, err = New(storage)
	utils.CrashOnError(err)
}

// Singleton returns the datastore instance.
func Singleton() DataStore {
	once.Do(initialize)
	return singleton
}
