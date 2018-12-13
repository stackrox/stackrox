package search

import (
	"github.com/stackrox/rox/central/alert/index"
	"github.com/stackrox/rox/central/alert/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Searcher provides search functionality on existing alerts
//go:generate mockgen-wrapper Searcher
type Searcher interface {
	SearchAlerts(q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawAlerts(q *v1.Query) ([]*storage.Alert, error)
	SearchListAlerts(q *v1.Query) ([]*storage.ListAlert, error)
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
