package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/sensorupgradeconfig/datastore/internal/store"
	"github.com/stackrox/stackrox/central/sensorupgradeconfig/datastore/internal/store/bolt"
	"github.com/stackrox/stackrox/central/sensorupgradeconfig/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once      sync.Once
	singleton DataStore
)

var (
	defaultConfig = &storage.SensorUpgradeConfig{
		EnableAutoUpgrade: true,
	}
)

func addDefaultConfigIfEmpty(d DataStore) error {
	ctx := sac.WithAllAccess(context.Background())
	currentConfig, err := d.GetSensorUpgradeConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check initial sensor upgrade config")
	}
	if currentConfig != nil {
		return nil
	}
	return d.UpsertSensorUpgradeConfig(ctx, defaultConfig)
}

func initialize() {
	var storage store.Store
	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(context.TODO(), globaldb.GetPostgres())
	} else {
		storage = bolt.New(globaldb.GetGlobalDB())
	}
	var err error
	singleton, err = New(storage)
	utils.CrashOnError(err)
	utils.Must(addDefaultConfigIfEmpty(singleton))
}

// Singleton returns the datastore instance.
func Singleton() DataStore {
	once.Do(initialize)
	return singleton
}
