package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/index"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/search"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to RoleBindingStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(q *v1.Query) ([]searchPkg.Result, error)
	SearchRoleBindings(q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawRoleBindings(q *v1.Query) ([]*storage.K8SRoleBinding, error)

	CountRoleBindings() (int, error)
	ListRoleBindings() ([]*storage.K8SRoleBinding, error)
	GetRoleBinding(id string) (*storage.K8SRoleBinding, bool, error)
	UpsertRoleBinding(request *storage.K8SRoleBinding) error
	RemoveRoleBinding(id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
	if err := d.buildIndex(); err != nil {
		return nil, errors.Wrapf(err, "failed to build index from existing store")
	}
	return d, nil
}
