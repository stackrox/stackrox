package search

import (
	"bitbucket.org/stack-rox/apollo/central/secret/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/blevesearch/bleve"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing secrets.
type Searcher interface {
	SearchSecrets(rawQuery *v1.RawQuery) ([]*v1.SearchResult, error)
	SearchRawSecrets(rawQuery *v1.RawQuery) ([]*v1.SecretAndRelationship, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, index bleve.Index) Searcher {
	return &searcherImpl{
		storage: storage,
		index:   index,
	}
}
