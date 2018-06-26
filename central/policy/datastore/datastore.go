package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/policy/index"
	"bitbucket.org/stack-rox/apollo/central/policy/search"
	"bitbucket.org/stack-rox/apollo/central/policy/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// DataStore is an intermediary to PolicyStorage.
type DataStore interface {
	SearchPolicies(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawPolicies(request *v1.ParsedSearchRequest) ([]*v1.Policy, error)

	GetPolicy(id string) (*v1.Policy, bool, error)
	GetPolicies() ([]*v1.Policy, error)

	AddPolicy(*v1.Policy) (string, error)
	UpdatePolicy(*v1.Policy) error
	RemovePolicy(id string) error
	RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error
	DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
}
