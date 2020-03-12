package search

import (
	"context"

	podIndexer "github.com/stackrox/rox/central/pod/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing pods
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(podIndexer podIndexer.Indexer) Searcher {
	return &searcherImpl{
		searcher: formatSearcher(podIndexer),
	}
}
