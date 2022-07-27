// This file was originally generated with
// //go:generate cp ../../../../central/sensorupgradeconfig/datastore/internal/store/store.go .

package legacy

import (
	"context"

	storage "github.com/stackrox/rox/generated/storage"
)

// Store provides access to the underlying data layer
type Store interface {
	Get(ctx context.Context) (*storage.SensorUpgradeConfig, bool, error)
	Upsert(ctx context.Context, sensorupgradeconfig *storage.SensorUpgradeConfig) error
}
