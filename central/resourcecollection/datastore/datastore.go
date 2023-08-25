package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store"
	pgStore "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
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
	AddCollection(ctx context.Context, collection *storage.ResourceCollection) (string, error)
	DeleteCollection(ctx context.Context, id string) error
	DryRunAddCollection(ctx context.Context, collection *storage.ResourceCollection) error
	// UpdateCollection updates the given collection object, and preserves createdAt and createdBy fields from stored collection
	UpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error
	DryRunUpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error
	// autocomplete workflow, maybe SearchResults? TODO ROX-12616
}

// QueryResolver provides functionality for resolving v1.Query objects from storage.ResourceCollection objects
type QueryResolver interface {
	ResolveCollectionQuery(ctx context.Context, collection *storage.ResourceCollection) (*v1.Query, error)
}

type supportedFieldKey struct {
	fieldLabel pkgSearch.FieldLabel
	labelType  bool
}

var (
	supportedFieldNames = map[string]supportedFieldKey{
		pkgSearch.Cluster.String():              {pkgSearch.Cluster, false},
		pkgSearch.ClusterLabel.String():         {pkgSearch.ClusterLabel, true},
		pkgSearch.Namespace.String():            {pkgSearch.Namespace, false},
		pkgSearch.NamespaceLabel.String():       {pkgSearch.NamespaceLabel, true},
		pkgSearch.NamespaceAnnotation.String():  {pkgSearch.NamespaceAnnotation, false},
		pkgSearch.DeploymentName.String():       {pkgSearch.DeploymentName, false},
		pkgSearch.DeploymentLabel.String():      {pkgSearch.DeploymentLabel, true},
		pkgSearch.DeploymentAnnotation.String(): {pkgSearch.DeploymentAnnotation, false},
	}
)

// GetSupportedFieldLabels returns a list of the supported search.FieldLabel values for resolving deployments for a collection
func GetSupportedFieldLabels() []pkgSearch.FieldLabel {
	var ret []pkgSearch.FieldLabel
	for _, label := range supportedFieldNames {
		ret = append(ret, label.fieldLabel)
	}
	return ret
}

// New returns a new instance of a DataStore and a QueryResolver.
func New(storage store.Store, searcher search.Searcher) (DataStore, QueryResolver, error) {
	ds := &datastoreImpl{
		storage:  storage,
		searcher: searcher,
	}

	if err := ds.initGraph(); err != nil {
		return nil, nil, err
	}
	return ds, ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) (DataStore, QueryResolver, error) {
	store := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(store, indexer)

	return New(store, searcher)
}
