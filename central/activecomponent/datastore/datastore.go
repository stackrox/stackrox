package datastore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/activecomponent/datastore/index"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/activecomponent/datastore/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/env"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
)

// DataStore is an intermediary to ActiveComponent storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, query *v1.Query) ([]pkgSearch.Result, error)
	SearchRawActiveComponents(ctx context.Context, q *v1.Query) ([]*storage.ActiveComponent, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ActiveComponent, bool, error)
	GetBatch(ctx context.Context, ids []string) ([]*storage.ActiveComponent, error)

	UpsertBatch(ctx context.Context, activeComponents []*storage.ActiveComponent) error
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

// NewForTestOnly returns a new instance of DataStore. TO BE USED FOR TESTING PURPOSES ONLY.
// To make this more explicit, we require passing a testing.T to this version.
func NewForTestOnly(t *testing.T, db *pgxpool.Pool) (DataStore, error) {
	testutils.MustBeInTest(t)

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return nil, nil
	}
	storage := postgres.New(db)
	indexer := postgres.NewIndexer(db)
	searcher := search.NewV2(storage, indexer)
	ds := &datastoreImpl{
		storage:       storage,
		graphProvider: nil,
		indexer:       indexer,
		searcher:      searcher,
	}

	return ds, nil
}
