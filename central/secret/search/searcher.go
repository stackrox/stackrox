package search

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing secrets.
//go:generate mockery -name=Searcher
type Searcher interface {
	SearchSecrets(*v1.Query) ([]*v1.SearchResult, error)
	SearchListSecrets(query *v1.Query) ([]*v1.ListSecret, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, index bleve.Index) Searcher {
	return &searcherImpl{
		storage: storage,
		index:   index,
	}
}
