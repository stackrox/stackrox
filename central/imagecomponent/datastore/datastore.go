package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	"github.com/stackrox/rox/central/imagecomponent/search"
	"github.com/stackrox/rox/central/imagecomponent/store"
	"github.com/stackrox/rox/central/ranking"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to ImageComponent storage.
//
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
func New(storage store.Store, searcher search.Searcher, risks riskDataStore.DataStore, ranker *ranking.Ranker) DataStore {
	ds := &datastoreImpl{
		storage:              storage,
		searcher:             searcher,
		risks:                risks,
		imageComponentRanker: ranker,
	}

	ds.initializeRankers()
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.NewV2(dbstore, indexer)
	riskStore := riskDataStore.GetTestPostgresDataStore(t, pool)
	return New(dbstore, searcher, riskStore, ranking.ComponentRanker())
}
