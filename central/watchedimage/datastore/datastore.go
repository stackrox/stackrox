package datastore

import (
	"context"

	"github.com/stackrox/rox/central/watchedimage/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore returns a datastore for watched images.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Exists(ctx context.Context, name string) (bool, error)
	UpsertWatchedImage(ctx context.Context, name string) error
	GetAllWatchedImages(ctx context.Context) ([]*storage.WatchedImage, error)
	UnwatchImage(ctx context.Context, name string) error
}

// New returns a new, ready-to-use DataStore.
func New(s store.Store) DataStore {
	ds := &dataStore{
		storage: s,
	}
	return ds
}
