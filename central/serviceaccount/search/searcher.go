package search

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/serviceaccount/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing service accounts.
//go:generate mockgen-wrapper Searcher
type Searcher interface {
	Search(query *v1.Query) ([]search.Result, error)
	SearchServiceAccounts(*v1.Query) ([]*v1.SearchResult, error)
	SearchRawServiceAccounts(*v1.Query) ([]*storage.ServiceAccount, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, index bleve.Index) Searcher {
	return &searcherImpl{
		storage: storage,
		index:   index,
	}
}
