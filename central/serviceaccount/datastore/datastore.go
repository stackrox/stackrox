package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	pgStore "github.com/stackrox/rox/central/serviceaccount/internal/store/postgres"
	"github.com/stackrox/rox/central/serviceaccount/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to ServiceAccountStorage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchRawServiceAccounts(ctx context.Context, q *v1.Query) ([]*storage.ServiceAccount, error)
	SearchServiceAccounts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)

	GetServiceAccount(ctx context.Context, id string) (*storage.ServiceAccount, bool, error)
	UpsertServiceAccount(ctx context.Context, request *storage.ServiceAccount) error
	RemoveServiceAccount(ctx context.Context, id string) error
}

// New returns a new instance of DataStore using the input store, and searcher.
func New(saStore store.Store, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  saStore,
		searcher: searcher,
	}
	return d, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	dbstore := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, searcher)
}
