package search

import (
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/image/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing alerts
//go:generate mockgen-wrapper Searcher
type Searcher interface {
	SearchImages(q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImages(q *v1.Query) ([]*storage.Image, error)
	SearchListImages(q *v1.Query) ([]*storage.ListImage, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, indexer index.Indexer) (Searcher, error) {
	ds := &searcherImpl{
		storage: storage,
		indexer: indexer,
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}
