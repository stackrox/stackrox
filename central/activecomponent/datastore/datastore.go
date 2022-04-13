package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/activecomponent/converter"
	"github.com/stackrox/stackrox/central/activecomponent/datastore/internal/store"
	"github.com/stackrox/stackrox/central/activecomponent/datastore/search"
	"github.com/stackrox/stackrox/central/activecomponent/index"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
)

// DataStore is an intermediary to ActiveComponent storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, query *v1.Query) ([]pkgSearch.Result, error)
	SearchRawActiveComponents(ctx context.Context, q *v1.Query) ([]*storage.ActiveComponent, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ActiveComponent, bool, error)
	GetBatch(ctx context.Context, ids []string) ([]*storage.ActiveComponent, error)

	UpsertBatch(ctx context.Context, activeComponents []*converter.CompleteActiveComponent) error
	DeleteBatch(ctx context.Context, ids ...string) error
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:       storage,
		graphProvider: graphProvider,
		indexer:       indexer,
		searcher:      searcher,
	}
	return ds
}
