package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/search"
	"github.com/stackrox/rox/central/policy/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to PolicyStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchPolicies(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawPolicies(ctx context.Context, q *v1.Query) ([]*storage.Policy, error)

	GetPolicy(ctx context.Context, id string) (*storage.Policy, bool, error)
	GetPolicies(ctx context.Context) ([]*storage.Policy, error)
	GetPolicyByName(ctx context.Context, name string) (*storage.Policy, bool, error)

	AddPolicy(context.Context, *storage.Policy) (string, error)
	UpdatePolicy(context.Context, *storage.Policy) error
	RemovePolicy(ctx context.Context, id string) error
	RenamePolicyCategory(ctx context.Context, request *v1.RenamePolicyCategoryRequest) error
	DeletePolicyCategory(ctx context.Context, request *v1.DeletePolicyCategoryRequest) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:    storage,
		indexer:    indexer,
		searcher:   searcher,
		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
}
