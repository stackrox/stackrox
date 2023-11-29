package search

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage  pgStore.Store
	searcher search.Searcher
}

func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (count int, err error) {
	return ds.searcher.Count(ctx, q)
}
