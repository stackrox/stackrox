package rocksdb

import (
	"context"

	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/rocksdb"
)

// NewFullStore returns the alert store interface
func NewFullStore(db *rocksdb.RocksDB) store.Store {
	return &fullStoreImpl{
		Store: New(db),
	}
}

// This implements the alert store interface
// it does not implement list alerts and instead converts alerts -> list alerts
type fullStoreImpl struct {
	Store
}

// ListAlert retrieves a single list alert
func (f *fullStoreImpl) ListAlert(ctx context.Context, id string) (*storage.ListAlert, bool, error) {
	alert, exists, err := f.Store.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}
	return convert.AlertToListAlert(alert), true, nil
}

// GetListAlerts returns list alert versions from the specified IDs
func (f *fullStoreImpl) GetListAlerts(ctx context.Context, ids []string) ([]*storage.ListAlert, []int, error) {
	// RocksDB MultiGet is similar performance to single gets so run single gets
	// also, this keeps memory pressure similar to previous runs
	var missingIndices []int
	listAlerts := make([]*storage.ListAlert, 0, len(ids))
	for idx, id := range ids {
		listAlert, exists, err := f.ListAlert(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			missingIndices = append(missingIndices, idx)
			continue
		}
		listAlerts = append(listAlerts, listAlert)
	}
	return listAlerts, missingIndices, nil
}

// Walk implements the walk interface of the store
func (f *fullStoreImpl) Walk(ctx context.Context, fn func(*storage.ListAlert) error) error {
	return f.Store.Walk(ctx, func(alert *storage.Alert) error {
		listAlert := convert.AlertToListAlert(alert)
		return fn(listAlert)
	})
}
