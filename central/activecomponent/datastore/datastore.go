package datastore

import (
	"context"

	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
)

// DataStore is an intermediary to ActiveComponent storage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ActiveComponent, bool, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.ActiveComponent, error)
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store) DataStore {
	ds := &datastoreImpl{
		storage:       storage,
		graphProvider: graphProvider,
	}
	return ds
}
