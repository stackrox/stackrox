package datastore

import (
	"context"

	"github.com/stackrox/rox/central/policycategoryedge/index"
	"github.com/stackrox/rox/central/policycategoryedge/search"
	"github.com/stackrox/rox/central/policycategoryedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to Policy Category edge storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.PolicyCategoryEdge, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.PolicyCategoryEdge, bool, error)
	GetAll(ctx context.Context) ([]*storage.PolicyCategoryEdge, error)

	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.PolicyCategoryEdge, error)

	UpsertMany(ctx context.Context, edges []*storage.PolicyCategoryEdge) error
	DeleteMany(ctx context.Context, ids ...string) error

	DeleteByQuery(ctx context.Context, q *v1.Query) error
}

// New returns a new instance of a DataStore.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
	return ds
}
