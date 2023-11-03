package search

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/index"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

type Searcher interface {
	Count(ctx context.Context, query *v1.Query) (int, error)
}

func New(storage pgStore.Store, indexer index.Indexer, search search.Searcher) *searcherImpl {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: search,
	}
}
