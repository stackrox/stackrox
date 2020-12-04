package datastore

import (
	"context"

	"github.com/stackrox/rox/central/imagecveedge/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
)

// DataStore is an intermediary to Image/CVE edge storage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Get(ctx context.Context, id string) (*storage.ImageCVEEdge, bool, error)
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store) (DataStore, error) {
	ds := &datastoreImpl{
		storage:       storage,
		graphProvider: graphProvider,
	}
	return ds, nil
}
