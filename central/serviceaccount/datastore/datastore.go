package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/serviceaccount/internal/index"
	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	"github.com/stackrox/rox/central/serviceaccount/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to ServiceAccountStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchRawServiceAccounts(ctx context.Context, q *v1.Query) ([]*storage.ServiceAccount, error)
	SearchServiceAccounts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)

	CountServiceAccounts(ctx context.Context) (int, error)
	ListServiceAccounts(ctx context.Context) ([]*storage.ServiceAccount, error)
	GetServiceAccount(ctx context.Context, id string) (*storage.ServiceAccount, bool, error)
	UpsertServiceAccount(ctx context.Context, request *storage.ServiceAccount) error
	RemoveServiceAccount(ctx context.Context, id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
	if err := d.buildIndex(); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil
}
