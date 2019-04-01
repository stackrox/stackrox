package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/k8srole/index"
	"github.com/stackrox/rox/central/rbac/k8srole/search"
	"github.com/stackrox/rox/central/rbac/k8srole/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to RoleStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(q *v1.Query) ([]searchPkg.Result, error)
	SearchRoles(q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawRoles(q *v1.Query) ([]*storage.K8SRole, error)

	CountRoles() (int, error)
	ListRoles() ([]*storage.K8SRole, error)
	GetRole(id string) (*storage.K8SRole, bool, error)
	UpsertRole(request *storage.K8SRole) error
	RemoveRole(id string) error
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
