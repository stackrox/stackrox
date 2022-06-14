package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/central/serviceaccount/internal/index"
	"github.com/stackrox/stackrox/central/serviceaccount/internal/store"
	"github.com/stackrox/stackrox/central/serviceaccount/internal/store/rocksdb"
	"github.com/stackrox/stackrox/central/serviceaccount/search"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	pkgRocksDB "github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/testutils"
)

// DataStore is an intermediary to ServiceAccountStorage.
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

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(saStore store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  saStore,
		indexer:  indexer,
		searcher: searcher,
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.ServiceAccount)))
	if err := d.buildIndex(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil
}

// NewForTestOnly returns a new instance of DataStore. TO BE USED FOR TESTING PURPOSES ONLY.
// To make this more explicit, we require passing a testing.T to this version.
func NewForTestOnly(t *testing.T, db *pkgRocksDB.RocksDB, bleveIndex bleve.Index) (DataStore, error) {
	testutils.MustBeInTest(t)
	saStore := rocksdb.New(db)
	indexer := index.New(bleveIndex)

	d := &datastoreImpl{
		storage:  saStore,
		indexer:  indexer,
		searcher: search.New(saStore, indexer),
	}

	if err := d.buildIndex(context.TODO()); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil
}
