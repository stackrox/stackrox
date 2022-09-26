package datastore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/nodecomponentcveedge/datastore/index"
	"github.com/stackrox/rox/central/nodecomponentcveedge/datastore/search"
	"github.com/stackrox/rox/central/nodecomponentcveedge/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to Component/CVE edge storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentCVEEdge, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.NodeComponentCVEEdge, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// New returns a new instance of a DataStore.
func New(storage postgres.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgres.New(pool)
	indexer := postgres.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, indexer, searcher)
}
