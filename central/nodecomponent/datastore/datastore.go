package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/nodecomponent/datastore/search"
	pgStore "github.com/stackrox/rox/central/nodecomponent/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to NodeComponent storage.
//
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
func New(storage pgStore.Store, searcher search.Searcher, risks riskDataStore.DataStore, ranker *ranking.Ranker) DataStore {
	ds := &datastoreImpl{
		storage:             storage,
		searcher:            searcher,
		risks:               risks,
		nodeComponentRanker: ranker,
	}

	ds.initializeRankers()
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	riskStore := riskDataStore.GetTestPostgresDataStore(t, pool)
	return New(dbstore, searcher, riskStore, ranking.NodeComponentRanker())
}
