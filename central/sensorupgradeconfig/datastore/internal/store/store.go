package store

import (
	"context"

	storage "github.com/stackrox/rox/generated/storage"
)

// Store provides access to the underlying data layer
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context) (*storage.SensorUpgradeConfig, bool, error)
	Upsert(ctx context.Context, sensorupgradeconfig *storage.SensorUpgradeConfig) error
}
