package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once      sync.Once
	singleton DataStore
)

func upgradeConfig(enabled bool, autoUpgradeAllowed storage.SensorAutoUpgrade) *storage.SensorUpgradeConfig {
	return &storage.SensorUpgradeConfig{
		EnableAutoUpgrade:  enabled,
		AutoUpgradeAllowed: autoUpgradeAllowed,
	}
}

func addDefaultConfigIfEmpty(d DataStore) error {
	ctx := sac.WithAllAccess(context.Background())
	currentConfig, err := d.GetSensorUpgradeConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check initial sensor upgrade config")
	}
	if currentConfig != nil && (!env.ManagedCentral.BooleanSetting() || !currentConfig.GetEnableAutoUpgrade()) {
		return nil
	}

	// Auto upgrade is disabled by default if managed central flag is set
	if env.ManagedCentral.BooleanSetting() {
		return d.UpsertSensorUpgradeConfig(ctx, upgradeConfig(false, storage.SensorAutoUpgrade_NOT_ALLOWED))
	}
	return d.UpsertSensorUpgradeConfig(ctx, upgradeConfig(true, storage.SensorAutoUpgrade_ALLOWED))
}

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
	utils.Must(addDefaultConfigIfEmpty(singleton))
}

// Singleton returns the datastore instance.
func Singleton() DataStore {
	once.Do(initialize)
	return singleton
}
