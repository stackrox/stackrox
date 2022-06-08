package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/index"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/store"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/store/rocksdb"
	"github.com/stackrox/rox/central/rbac/k8srole/search"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgRocksDB "github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/utils"
)

// DataStore is an intermediary to RoleStorage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchRoles(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawRoles(ctx context.Context, q *v1.Query) ([]*storage.K8SRole, error)

	GetRole(ctx context.Context, id string) (*storage.K8SRole, bool, error)
	UpsertRole(ctx context.Context, request *storage.K8SRole) error
	RemoveRole(ctx context.Context, id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(k8sRoleStore store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  k8sRoleStore,
		indexer:  indexer,
		searcher: searcher,
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.K8sRole)))
	if err := d.buildIndex(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil
}

// NewForTestOnly returns a new instance of DataStore. TO BE USED FOR TESTING PURPOSES ONLY.
// To make this more explicit, we require passing a testing.T to this version.
func NewForTestOnly(t *testing.T, db *pkgRocksDB.RocksDB, bleveIndex bleve.Index) (DataStore, error) {
	utils.MustBeInTest(t)
	k8sRoleStore := rocksdb.New(db)
	indexer := index.New(bleveIndex)

	d := &datastoreImpl{
		storage:  k8sRoleStore,
		indexer:  indexer,
		searcher: search.New(k8sRoleStore, indexer),
	}

	if err := d.buildIndex(context.TODO()); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil
}
