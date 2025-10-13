package datastore

import (
	"context"

	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for the sensor upgrade config.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetSensorUpgradeConfig(context.Context) (*storage.SensorUpgradeConfig, error)
	UpsertSensorUpgradeConfig(context.Context, *storage.SensorUpgradeConfig) error
}

// New returns a new, ready-to-use, datastore.
func New(store store.Store) DataStore {
	return &dataStore{store: store}
}
