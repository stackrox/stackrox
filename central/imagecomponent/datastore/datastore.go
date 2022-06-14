package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/imagecomponent/index"
	"github.com/stackrox/stackrox/central/imagecomponent/search"
	"github.com/stackrox/stackrox/central/imagecomponent/store"
	"github.com/stackrox/stackrox/central/ranking"
	riskDataStore "github.com/stackrox/stackrox/central/risk/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
)

// DataStore is an intermediary to ImageComponent storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchImageComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImageComponents(ctx context.Context, q *v1.Query) ([]*storage.ImageComponent, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ImageComponent, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.ImageComponent, error)
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store, indexer index.Indexer, searcher search.Searcher, risks riskDataStore.DataStore, ranker *ranking.Ranker) (DataStore, error) {
	ds := &datastoreImpl{
		storage:              storage,
		indexer:              indexer,
		searcher:             searcher,
		graphProvider:        graphProvider,
		risks:                risks,
		imageComponentRanker: ranker,
	}

	ds.initializeRankers()
	return ds, nil
}
