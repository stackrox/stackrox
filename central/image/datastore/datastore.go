package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/image/index"
	"bitbucket.org/stack-rox/apollo/central/image/search"
	"bitbucket.org/stack-rox/apollo/central/image/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// DataStore is an intermediary to AlertStorage.
type DataStore interface {
	SearchListImages(request *v1.ParsedSearchRequest) ([]*v1.ListImage, error)
	ListImage(sha string) (*v1.ListImage, bool, error)
	ListImages() ([]*v1.ListImage, error)

	SearchImages(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawImages(request *v1.ParsedSearchRequest) ([]*v1.Image, error)

	GetImages() ([]*v1.Image, error)
	CountImages() (int, error)
	GetImage(sha string) (*v1.Image, bool, error)
	GetImagesBatch(shas []string) ([]*v1.Image, error)
	UpsertDedupeImage(image *v1.Image) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
}
