package datastore

import (
	"context"

	"github.com/stackrox/rox/central/policycategory/index"
	"github.com/stackrox/rox/central/policycategory/search"
	"github.com/stackrox/rox/central/policycategory/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to policy category storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchRawPolicyCategories(ctx context.Context, q *v1.Query) ([]*storage.PolicyCategory, error)
	SearchPolicyCategories(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)

	GetPolicyCategory(ctx context.Context, id string) (*storage.PolicyCategory, bool, error)
	GetAllPolicyCategories(ctx context.Context) ([]*storage.PolicyCategory, error)

	AddPolicyCategory(context.Context, *storage.PolicyCategory) (*storage.PolicyCategory, error)
	RenamePolicyCategory(ctx context.Context, id, newName string) error
	DeletePolicyCategory(ctx context.Context, id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}

	if err := ds.buildIndex(); err != nil {
		panic("unable to load search index for policy categories")
	}
	return ds
}

// newWithoutDefaults should be used only for testing purposes.
func newWithoutDefaults(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
}
