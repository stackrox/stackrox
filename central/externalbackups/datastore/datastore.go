package datastore

import (
	"context"

	"github.com/stackrox/rox/central/externalbackups/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the entry point for modifying External Backup data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	ForEachBackup(ctx context.Context, fn func(obj *storage.ExternalBackup) error) error
	GetBackup(ctx context.Context, id string) (*storage.ExternalBackup, bool, error)
	UpsertBackup(ctx context.Context, backup *storage.ExternalBackup) error
	RemoveBackup(ctx context.Context, id string) error
}

// New returns an instance of DataStore.
func New(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}
