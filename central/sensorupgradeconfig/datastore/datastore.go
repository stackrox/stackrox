package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/sensorupgradeconfig/datastore/internal/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
)

// DataStore is the datastore for the sensor upgrade config.
//go:generate mockgen-wrapper
type DataStore interface {
	GetSensorUpgradeConfig(context.Context) (*storage.SensorUpgradeConfig, error)
	UpsertSensorUpgradeConfig(context.Context, *storage.SensorUpgradeConfig) error

	AutoTriggerSetting() *concurrency.Flag
}

// New returns a new, ready-to-use, datastore.
func New(store store.Store) (DataStore, error) {
	ds := &dataStore{store: store}
	if err := ds.initialize(); err != nil {
		return nil, err
	}
	return ds, nil
}
