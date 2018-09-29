package datastore

import (
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/image/search"
	"github.com/stackrox/rox/central/image/store"
	"github.com/stackrox/rox/generated/api/v1"
)

// DataStore is an intermediary to AlertStorage.
//go:generate mockery -name=DataStore
type DataStore interface {
	SearchListImages(q *v1.Query) ([]*v1.ListImage, error)
	ListImage(sha string) (*v1.ListImage, bool, error)
	ListImages() ([]*v1.ListImage, error)

	SearchImages(q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImages(q *v1.Query) ([]*v1.Image, error)

	GetImages() ([]*v1.Image, error)
	CountImages() (int, error)
	GetImage(sha string) (*v1.Image, bool, error)
	GetImagesBatch(shas []string) ([]*v1.Image, error)
	UpsertImage(image *v1.Image) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
}
