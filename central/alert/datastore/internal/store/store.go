package store

import (
	"context"

	"github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/alert/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	ListAlert(ctx context.Context, id string) (*storage.ListAlert, bool, error)
	GetListAlerts(context.Context, []string) ([]*storage.ListAlert, []int, error)

	Walk(ctx context.Context, fn func(*storage.ListAlert) error) error
	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.Alert, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Alert, []int, error)
	Upsert(ctx context.Context, alert *storage.Alert) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}

// GenStore provides the interface for generated effective store code.
// This is a subset of what Store defines, but which is enough to implement
// the full functionality
type GenStore interface {
	Walk(ctx context.Context, fn func(*storage.Alert) error) error
	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.Alert, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Alert, []int, error)
	Upsert(ctx context.Context, alert *storage.Alert) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}

// NewFullStore returns the alert store interface
func NewFullStore(store rocksdb.Store) Store {
	return &fullStoreImpl{
		GenStore: store,
	}
}

// NewFullPgStore returns the alert store interface
func NewFullPgStore(store postgres.Store) Store {
	return &fullStoreImpl{
		GenStore: store,
	}
}

// This implements the alert store interface
// it does not implement list alerts and instead converts alerts -> list alerts
type fullStoreImpl struct {
	GenStore
}

// ListAlert retrieves a single list alert
func (f *fullStoreImpl) ListAlert(ctx context.Context, id string) (*storage.ListAlert, bool, error) {
	alert, exists, err := f.GenStore.Get(ctx, id)
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
	return f.GenStore.Walk(ctx, func(alert *storage.Alert) error {
		listAlert := convert.AlertToListAlert(alert)
		return fn(listAlert)
	})
}
