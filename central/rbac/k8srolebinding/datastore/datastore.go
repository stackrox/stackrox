package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/index"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	pgStore "github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store/postgres"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store/rocksdb"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/search"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	pkgRocksDB "github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
)

// DataStore is an intermediary to RoleBindingStorage.
//
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
func New(k8sRoleBindingStore store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  k8sRoleBindingStore,
		indexer:  indexer,
		searcher: searcher,
	}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.K8sRoleBinding)))
	if err := d.buildIndex(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil
}

// NewForTestOnly returns a new instance of DataStore. TO BE USED FOR TESTING PURPOSES ONLY.
// To make this more explicit, we require passing a testing.T to this version.
func NewForTestOnly(t *testing.T, db *pkgRocksDB.RocksDB, bleveIndex bleve.Index) (DataStore, error) {
	testutils.MustBeInTest(t)
	k8sRoleBindingStore := rocksdb.New(db)
	indexer := index.New(bleveIndex)
	searcher := search.New(k8sRoleBindingStore, indexer)

	return New(k8sRoleBindingStore, indexer, searcher)
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool *postgres.DB) (DataStore, error) {
	dbstore := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, indexer, searcher)
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *pkgRocksDB.RocksDB, bleveIndex bleve.Index) (DataStore, error) {
	return NewForTestOnly(t, rocksengine, bleveIndex)
}
