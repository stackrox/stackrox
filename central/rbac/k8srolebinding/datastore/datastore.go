package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/index"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store/rocksdb"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgRocksDB "github.com/stackrox/rox/pkg/rocksdb"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
)

// DataStore is an intermediary to RoleBindingStorage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchRoleBindings(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawRoleBindings(ctx context.Context, q *v1.Query) ([]*storage.K8SRoleBinding, error)

	GetRoleBinding(ctx context.Context, id string) (*storage.K8SRoleBinding, bool, error)
	UpsertRoleBinding(ctx context.Context, request *storage.K8SRoleBinding) error
	RemoveRoleBinding(ctx context.Context, id string) error
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

// NewForTestOnly returns a new instance of DataStore. TO BE USED FOR TESTING PURPOSES ONLY.
// To make this more explicit, we require passing a testing.T to this version.
func NewForTestOnly(t *testing.T, db *pkgRocksDB.RocksDB, bleveIndex bleve.Index) (DataStore, error) {
	testutils.MustBeInTest(t)
	storage := rocksdb.New(db)
	indexer := index.New(bleveIndex)

	d := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: search.New(storage, indexer),
	}

	if err := d.buildIndex(); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil
}
