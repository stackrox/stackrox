package search

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

type searcherImpl struct {
	storage pgStore.Store
}

func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (count int, err error) {
	return ds.storage.Count(ctx, q)
}
