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
	searchPkg "github.com/stackrox/rox/pkg/search"
)

//go:generate mockgen-wrapper

// DataStore is the entry point for modifying Collection data.
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
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
	UpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error
	// autocomplete workflow, maybe SearchResults? TODO ROX-12616
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
