package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/imagecveedge/search"
	"github.com/stackrox/stackrox/central/imagecveedge/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
)

// DataStore is an intermediary to Image/CVE edge storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEEdge, error)
	Get(ctx context.Context, id string) (*storage.ImageCVEEdge, bool, error)
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		graphProvider: graphProvider,
		storage:       storage,
		searcher:      searcher,
	}
}
