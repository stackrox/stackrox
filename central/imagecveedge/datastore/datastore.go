package datastore

import (
	"context"

	"github.com/stackrox/rox/central/imagecveedge/search"
	"github.com/stackrox/rox/central/imagecveedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to Image/CVE edge storage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEEdge, error)
	Get(ctx context.Context, id string) (*storage.ImageCVEEdge, bool, error)
}

// New returns a new instance of a DataStore.
func New(storage store.Store, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		searcher: searcher,
	}
}
