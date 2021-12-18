package datastore

import (
	"context"

	"github.com/stackrox/rox/central/nodecveedge/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
)

// DataStore is an intermediary to Node/CVE edge storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Get(ctx context.Context, id string) (*storage.NodeCVEEdge, bool, error)
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store) DataStore {
	return &datastoreImpl{
		storage:       storage,
		graphProvider: graphProvider,
	}
}
