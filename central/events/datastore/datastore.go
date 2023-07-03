package datastore

import (
	"context"

	"github.com/stackrox/rox/central/events/datastore/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore provides an interface to handle events.
type DataStore interface {
	GetEvents(ctx context.Context) ([]*storage.Event, error)
	AddEvent(ctx context.Context, event *storage.Event) error
}

// New creates a new DataStore.
func New(storage store.Store) DataStore {
	return &datastoreImpl{
		store: storage,
	}
}
