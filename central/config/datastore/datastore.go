package datastore

import (
	"context"

	"github.com/stackrox/rox/central/config/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the entry point for modifying Config data.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	GetConfig(context.Context) (*storage.Config, error)
	UpdateConfig(context.Context, *storage.Config) error
}

// New returns an instance of DataStore.
func New(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

type datastoreImpl struct {
	store store.Store
}

// GetConfig returns Central's config
func (d *datastoreImpl) GetConfig(context.Context) (*storage.Config, error) {
	return d.store.GetConfig()
}

// UpdateConfig updates Central's config
func (d *datastoreImpl) UpdateConfig(_ context.Context, config *storage.Config) error {
	return d.store.UpdateConfig(config)
}
