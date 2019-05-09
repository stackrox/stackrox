package datastore

import (
	"context"

	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/search"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/globaldb"
	flowStore "github.com/stackrox/rox/central/networkflow/store"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	whitelistDataStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to AlertStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error)
	SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error)

	ListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error)
	ListDeployments(ctx context.Context) ([]*storage.ListDeployment, error)

	GetDeployment(ctx context.Context, id string) (*storage.Deployment, bool, error)
	GetDeployments(ctx context.Context) ([]*storage.Deployment, error)
	CountDeployments(ctx context.Context) (int, error)
	// UpsertDeployment adds or updates a deployment. It should only be called the caller
	// is okay with inserting the passed deployment if it doesn't already exist in the store.
	// If you only want to update a deployment if it exists, call UpdateDeployment below.
	UpsertDeployment(ctx context.Context, deployment *storage.Deployment) error
	// UpdateDeployment updates a deployment, erroring out if it doesn't exist.
	UpdateDeployment(ctx context.Context, deployment *storage.Deployment) error
	RemoveDeployment(ctx context.Context, clusterID, id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher, processDataStore processDataStore.DataStore, whitelistDataStore whitelistDataStore.DataStore, networkFlowStore flowStore.ClusterStore) DataStore {
	return &datastoreImpl{
		deploymentStore:    storage,
		deploymentIndexer:  indexer,
		deploymentSearcher: searcher,
		processDataStore:   processDataStore,
		whitelistDataStore: whitelistDataStore,
		networkFlowStore:   networkFlowStore,
		keyedMutex:         concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
}
