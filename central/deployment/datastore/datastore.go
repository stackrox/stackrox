package datastore

import (
	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/search"
	"github.com/stackrox/rox/central/deployment/store"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to AlertStorage.
//go:generate mockery -name=DataStore
type DataStore interface {
	Search(q *v1.Query) ([]pkgSearch.Result, error)
	SearchDeployments(q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawDeployments(q *v1.Query) ([]*v1.Deployment, error)
	SearchListDeployments(q *v1.Query) ([]*v1.ListDeployment, error)

	ListDeployment(id string) (*v1.ListDeployment, bool, error)
	ListDeployments() ([]*v1.ListDeployment, error)

	GetDeployment(id string) (*v1.Deployment, bool, error)
	GetDeployments() ([]*v1.Deployment, error)
	CountDeployments() (int, error)
	// UpsertDeployment adds or updates a deployment. It should only be called the caller
	// is okay with inserting the passed deployment if it doesn't already exist in the store.
	// If you only want to update a deployment if it exists, call UpdateDeployment below.
	UpsertDeployment(deployment *v1.Deployment) error
	// UpdateDeployment updates a deployment, erroring out if it doesn't exist.
	UpdateDeployment(deployment *v1.Deployment) error
	RemoveDeployment(id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher, processDataStore processDataStore.DataStore) DataStore {
	return &datastoreImpl{
		deploymentStore:    storage,
		deploymentIndexer:  indexer,
		deploymentSearcher: searcher,
		processDataStore:   processDataStore,
	}
}
