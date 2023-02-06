package datastore

import (
	"context"

	"github.com/stackrox/rox/central/serviceidentities/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for serviceidentities keys.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetServiceIdentities(context.Context) ([]*storage.ServiceIdentity, error)
	AddServiceIdentity(ctx context.Context, identity *storage.ServiceIdentity) error
}

// New returns a new DataStore instance.
func New(storage store.Store) DataStore {
	return &dataStoreImpl{
		storage: storage,
	}
}
