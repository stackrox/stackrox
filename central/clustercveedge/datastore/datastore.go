package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/clustercveedge/index"
	"github.com/stackrox/stackrox/central/clustercveedge/search"
	"github.com/stackrox/stackrox/central/clustercveedge/store"
	"github.com/stackrox/stackrox/central/cve/converter"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
)

// DataStore is an intermediary to Cluster/CVE edge storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ClusterCVEEdge, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ClusterCVEEdge, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.ClusterCVEEdge, error)

	Upsert(ctx context.Context, cves ...converter.ClusterCVEParts) error
	Delete(ctx context.Context, ids ...string) error
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:       storage,
		indexer:       indexer,
		searcher:      searcher,
		graphProvider: graphProvider,
	}
	return ds, nil
}
