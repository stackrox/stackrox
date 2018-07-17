package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/deployment/index"
	"bitbucket.org/stack-rox/apollo/central/deployment/search"
	"bitbucket.org/stack-rox/apollo/central/deployment/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// DataStore is an intermediary to AlertStorage.
type DataStore interface {
	SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, error)
	SearchListDeployments(request *v1.ParsedSearchRequest) ([]*v1.ListDeployment, error)

	ListDeployment(id string) (*v1.ListDeployment, bool, error)
	ListDeployments() ([]*v1.ListDeployment, error)

	GetDeployment(id string) (*v1.Deployment, bool, error)
	GetDeployments() ([]*v1.Deployment, error)
	CountDeployments() (int, error)
	AddDeployment(alert *v1.Deployment) error
	UpdateDeployment(alert *v1.Deployment) error
	RemoveDeployment(id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
}
