package datastore

import (
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/api/v1"
)

// DataStore is an intermediary to PolicyStorage.
type DataStore interface {
	SearchProcessIndicators(q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawProcessIndicators(q *v1.Query) ([]*v1.ProcessIndicator, error)

	GetProcessIndicator(id string) (*v1.ProcessIndicator, bool, error)
	GetProcessIndicators() ([]*v1.ProcessIndicator, error)
	AddProcessIndicator(*v1.ProcessIndicator) error
	AddProcessIndicators(...*v1.ProcessIndicator) error
	RemoveProcessIndicator(id string) error
	RemoveProcessIndicatorsByDeployment(id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
}
