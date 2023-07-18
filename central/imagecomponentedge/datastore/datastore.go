package datastore

import (
	"context"

	"github.com/stackrox/rox/central/imagecomponentedge/search"
	"github.com/stackrox/rox/central/imagecomponentedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to Image/Component edge storage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageComponentEdge, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ImageComponentEdge, bool, error)
	Count(ctx context.Context) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.ImageComponentEdge, error)
}

// New returns a new instance of a DataStore.
func New(storage store.Store, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:  storage,
		searcher: searcher,
	}
	return ds, nil
}
