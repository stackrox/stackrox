package datastore

import (
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/image/search"
	"github.com/stackrox/rox/central/image/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to AlertStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	SearchListImages(q *v1.Query) ([]*storage.ListImage, error)
	ListImage(sha string) (*storage.ListImage, bool, error)
	ListImages() ([]*storage.ListImage, error)

	Search(q *v1.Query) ([]searchPkg.Result, error)
	SearchImages(q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImages(q *v1.Query) ([]*storage.Image, error)

	GetImages() ([]*storage.Image, error)
	CountImages() (int, error)
	GetImage(sha string) (*storage.Image, bool, error)
	GetImagesBatch(shas []string) ([]*storage.Image, error)
	UpsertImage(image *storage.Image) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
}
