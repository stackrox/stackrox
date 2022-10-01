package datastore

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/resourcecollection/datastore/index"
	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
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
	GetBatch(ctx context.Context, id []string) ([]*storage.ResourceCollection, error)

	// AddCollection(ctx context.Context, collection *storage.ResourceCollection) (string, error) TODO ROX-12612
	// AddCollectionDryRun(ctx context.Context, collection *storage.ResourceCollection) error TODO ROX-12615
	// UpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error TODO ROX-12614
	// DeleteCollection(ctx context.Context, id string) error TODO ROX-12613
	// autocomplete workflow, maybe SearchResults? TODO ROX-12616
}

// New returns a new instance of a DataStore.
func New(storage postgres.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}

	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *pgPkg.Postgres) (DataStore, error) {
	if t == nil {
		// this function should only be used for testing purposes
		return nil, errors.New("TestPostgresDataStore should only be used within testing context")
	}
	dbstore := postgres.New(pool)
	indexer := postgres.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, indexer, searcher), nil
}
