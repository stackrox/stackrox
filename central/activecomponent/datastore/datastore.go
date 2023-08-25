package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/activecomponent/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/activecomponent/datastore/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
)

// DataStore is an intermediary to ActiveComponent storage.
//
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
func New(storage store.Store, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:  storage,
		searcher: searcher,
	}
	return ds
}

// NewForTestOnly returns a new instance of DataStore. TO BE USED FOR TESTING PURPOSES ONLY.
// To make this more explicit, we require passing a testing.T to this version.
func NewForTestOnly(t *testing.T, db postgres.DB) (DataStore, error) {
	testutils.MustBeInTest(t)

	storage := pgStore.New(db)
	searcher := search.NewV2(storage, pgStore.NewIndexer(db))
	ds := &datastoreImpl{
		storage:  storage,
		searcher: searcher,
	}

	return ds, nil
}
