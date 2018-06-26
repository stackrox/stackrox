package search

import (
	"bitbucket.org/stack-rox/apollo/central/policy/index"
	"bitbucket.org/stack-rox/apollo/central/policy/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing alerts
type Searcher interface {
	SearchPolicies(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawPolicies(request *v1.ParsedSearchRequest) ([]*v1.Policy, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, indexer index.Indexer) (Searcher, error) {
	ds := &searcherImpl{
		storage: storage,
		indexer: indexer,
	}
	if err := ds.loadDefaults(); err != nil {
		return nil, err
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}
