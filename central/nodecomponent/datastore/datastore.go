package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/nodecomponent/datastore/index"
	"github.com/stackrox/stackrox/central/nodecomponent/datastore/search"
	"github.com/stackrox/stackrox/central/nodecomponent/datastore/store/postgres"
	"github.com/stackrox/stackrox/central/ranking"
	riskDataStore "github.com/stackrox/stackrox/central/risk/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
)

// DataStore is an intermediary to NodeComponent storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchNodeComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawNodeComponents(ctx context.Context, q *v1.Query) ([]*storage.NodeComponent, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.NodeComponent, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.NodeComponent, error)
}

// New returns a new instance of a DataStore.
func New(storage postgres.Store, indexer index.Indexer, searcher search.Searcher, risks riskDataStore.DataStore, ranker *ranking.Ranker) (DataStore, error) {
	ds := &datastoreImpl{
		storage:             storage,
		indexer:             indexer,
		searcher:            searcher,
		risks:               risks,
		nodeComponentRanker: ranker,
	}

	ds.initializeRankers()
	return ds, nil
}
