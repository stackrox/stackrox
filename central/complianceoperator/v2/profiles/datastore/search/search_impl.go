package search

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type searcherImpl struct {
	storage pgStore.Store
}

// Count returns the number of profiles from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if ok, err := complianceSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if !ok {
		return 0, sac.ErrResourceAccessDenied
	}
	return ds.storage.Count(ctx, q)
}
