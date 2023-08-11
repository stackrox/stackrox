package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	pgStore "github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store/postgres"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
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
	GetManyRoleBindings(ctx context.Context, ids []string) ([]*storage.K8SRoleBinding, []int, error)
	UpsertRoleBinding(ctx context.Context, request *storage.K8SRoleBinding) error
	RemoveRoleBinding(ctx context.Context, id string) error
}

// New returns a new instance of DataStore using the input store, and searcher.
func New(k8sRoleBindingStore store.Store, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  k8sRoleBindingStore,
		searcher: searcher,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, searcher)
}
