package search

import (
	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing alerts
type Searcher interface {
	SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, error)
	SearchListDeployments(request *v1.ParsedSearchRequest) ([]*v1.ListDeployment, error)
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
