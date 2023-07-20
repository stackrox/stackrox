package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/rbac/k8srole/internal/store"
	pgStore "github.com/stackrox/rox/central/rbac/k8srole/internal/store/postgres"
	"github.com/stackrox/rox/central/rbac/k8srole/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to RoleStorage.
//
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

// New returns a new instance of DataStore using the input store, and searcher.
func New(k8sRoleStore store.Store, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  k8sRoleStore,
		searcher: searcher,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool)
	searcher := search.New(dbstore, pgStore.NewIndexer(pool))
	return New(dbstore, searcher)
}
