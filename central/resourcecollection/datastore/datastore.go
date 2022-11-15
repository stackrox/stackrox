package datastore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/resourcecollection/datastore/index"
	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

//go:generate mockgen-wrapper

// DataStore is the entry point for modifying Collection data.
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchCollections(ctx context.Context, q *v1.Query) ([]*storage.ResourceCollection, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ResourceCollection, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetMany(ctx context.Context, id []string) ([]*storage.ResourceCollection, error)

	// AddCollection adds the given collection object and populates the `Id` field on the object
	AddCollection(ctx context.Context, collection *storage.ResourceCollection) error
	DeleteCollection(ctx context.Context, id string) error
	DryRunAddCollection(ctx context.Context, collection *storage.ResourceCollection) error
	// UpdateCollection updates the given collection object, and preserves createdAt and createdBy fields from stored collection
	UpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error
	DryRunUpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error
	ResolveListDeployments(ctx context.Context, collection *storage.ResourceCollection, pagination *v1.Pagination) ([]*storage.ListDeployment, error)
	// autocomplete workflow, maybe SearchResults? TODO ROX-12616

	// ResolveCollectionQuery exported exclusively for testing purposes, should be hidden once e2e tests go in
	// ResolveCollectionQuery(ctx context.Context, collection *storage.ResourceCollection) (*v1.Query, error)
}

var (
	supportedFieldNames = map[string]pkgSearch.FieldLabel{
		pkgSearch.Cluster.String():              pkgSearch.Cluster,
		pkgSearch.ClusterLabel.String():         pkgSearch.ClusterLabel,
		pkgSearch.Namespace.String():            pkgSearch.Namespace,
		pkgSearch.NamespaceLabel.String():       pkgSearch.NamespaceLabel,
		pkgSearch.NamespaceAnnotation.String():  pkgSearch.NamespaceAnnotation,
		pkgSearch.DeploymentName.String():       pkgSearch.DeploymentName,
		pkgSearch.DeploymentLabel.String():      pkgSearch.DeploymentLabel,
		pkgSearch.DeploymentAnnotation.String(): pkgSearch.DeploymentAnnotation,
	}
)

// GetSupportedFieldLabels returns a list of the supported search.FieldLabel values for resolving deployments for a collection
func GetSupportedFieldLabels() []pkgSearch.FieldLabel {
	var ret []pkgSearch.FieldLabel
	for _, label := range supportedFieldNames {
		ret = append(ret, label)
	}
	return ret
}

// New returns a new instance of a DataStore.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}

	if err := ds.initGraph(); err != nil {
		return nil, err
	}
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	if t == nil {
		// this function should only be used for testing purposes
		return nil, errors.New("TestPostgresDataStore should only be used within testing context")
	}
	dbstore := postgres.New(pool)
	indexer := postgres.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, indexer, searcher)
}
