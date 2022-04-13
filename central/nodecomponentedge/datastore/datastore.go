package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/nodecomponentedge/index"
	"github.com/stackrox/stackrox/central/nodecomponentedge/search"
	"github.com/stackrox/stackrox/central/nodecomponentedge/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
)

// DataStore is an intermediary to Node/Component edge storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentEdge, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.NodeComponentEdge, bool, error)
	Count(ctx context.Context) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.NodeComponentEdge, error)
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:       storage,
		indexer:       indexer,
		searcher:      searcher,
		graphProvider: graphProvider,
	}
}
