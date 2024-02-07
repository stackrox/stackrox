package search

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/integration/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

type searcherImpl struct {
	storage pgStore.Store
}

// Count returns the number of integrations from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, q)
}
