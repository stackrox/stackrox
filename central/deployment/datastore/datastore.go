package datastore

import (
	"context"

	"github.com/blevesearch/bleve"
	"github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/deployment/datastore/internal/search"
	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/store"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	pwDS "github.com/stackrox/rox/central/processwhitelist/datastore"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to AlertStorage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error)
	SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error)

	ListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error)

	GetDeployment(ctx context.Context, id string) (*storage.Deployment, bool, error)
	GetDeployments(ctx context.Context, ids []string) ([]*storage.Deployment, error)
	CountDeployments(ctx context.Context) (int, error)
	// UpsertDeployment adds or updates a deployment. It should only be called the caller
	// is okay with inserting the passed deployment if it doesn't already exist in the store.
	// If you only want to update a deployment if it exists, call UpdateDeployment below.
	UpsertDeployment(ctx context.Context, deployment *storage.Deployment) error
	// UpdateDeployment updates a deployment, erroring out if it doesn't exist.
	UpdateDeployment(ctx context.Context, deployment *storage.Deployment) error
	RemoveDeployment(ctx context.Context, clusterID, id string) error

	GetImagesForDeployment(ctx context.Context, deployment *storage.Deployment) ([]*storage.Image, error)
}

// New returns a new instance of DataStore using the input DB and index.
func New(db *bbolt.DB, bleveIndex bleve.Index, images imageDS.DataStore, indicators piDS.DataStore, whitelists pwDS.DataStore, networkFlows nfDS.ClusterDataStore, risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache) (DataStore, error) {
	storage, err := store.New(db)
	if err != nil {
		return nil, err
	}
	indexer := index.New(bleveIndex)
	searcher := search.New(storage, indexer)

	ds, err := newDatastoreImpl(
		storage,
		indexer,
		searcher,
		images,
		indicators,
		whitelists,
		networkFlows,
		risks,
		deletedDeploymentCache)

	if err != nil {
		return nil, err
	}

	if err := ds.initializeRanker(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize ranker")
	}

	return ds, nil
}
