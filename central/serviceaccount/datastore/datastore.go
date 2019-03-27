package datastore

import (
	"fmt"

	"github.com/stackrox/rox/central/serviceaccount/index"
	"github.com/stackrox/rox/central/serviceaccount/search"
	"github.com/stackrox/rox/central/serviceaccount/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to ServiceAccountStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(q *v1.Query) ([]searchPkg.Result, error)
	SearchRawServiceAccounts(q *v1.Query) ([]*storage.ServiceAccount, error)
	SearchServiceAccounts(q *v1.Query) ([]*v1.SearchResult, error)

	CountServiceAccounts() (int, error)
	ListServiceAccounts() ([]*storage.ServiceAccount, error)
	GetServiceAccount(id string) (*storage.ServiceAccount, bool, error)
	UpsertServiceAccount(request *storage.ServiceAccount) error
	RemoveServiceAccount(id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
	if err := d.buildIndex(); err != nil {
		return nil, fmt.Errorf("failed to build index from existing store: %s", err)
	}
	return d, nil
}
