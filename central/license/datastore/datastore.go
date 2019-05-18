package datastore

import (
	"context"

	"github.com/stackrox/rox/central/license/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for license keys.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	ListLicenseKeys(ctx context.Context) ([]*storage.StoredLicenseKey, error)
	UpsertLicenseKeys(ctx context.Context, keys []*storage.StoredLicenseKey) error
	DeleteLicenseKey(ctx context.Context, licenseID string) error
}

// New returns a new DataStore instance.
func New(storage store.Store) DataStore {
	return &dataStoreImpl{
		storage: storage,
	}
}
