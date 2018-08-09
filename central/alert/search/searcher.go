package search

import (
	"github.com/stackrox/rox/central/alert/index"
	"github.com/stackrox/rox/central/alert/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing alerts
type Searcher interface {
	SearchAlerts(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, error)
	SearchListAlerts(request *v1.ParsedSearchRequest) ([]*v1.ListAlert, error)
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
