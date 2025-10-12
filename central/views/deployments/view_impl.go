package deployments

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	queryTimeout = env.PostgresVMStatementTimeout.DurationSetting()
)

type deploymentViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *deploymentViewImpl) Get(ctx context.Context, query *v1.Query) ([]DeploymentCore, error) {
	if err := common.ValidateQuery(query); err != nil {
		return nil, err
	}

	var err error
	// Update the sort options to use aggregations if necessary as we are grouping by CVEs
	query = common.UpdateSortAggs(query)
	query, err = common.WithSACFilter(ctx, resources.Deployment, query)
	if err != nil {
		return nil, err
	}
	query = withSelectQuery(query)

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]DeploymentCore, 0, paginated.GetLimit(query.GetPagination().GetLimit(), 100))
	err = pgSearch.RunSelectRequestForSchemaFn[deploymentResponse](queryCtx, v.db, v.schema, query, func(r *deploymentResponse) error {
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func withSelectQuery(query *v1.Query) *v1.Query {
	cloned := query.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.DeploymentID).Distinct().Proto(),
	}

	if common.IsSortBySeverityCounts(cloned) {
		cloned.GroupBy = &v1.QueryGroupBy{
			Fields: []string{search.DeploymentID.String()},
		}
		cloned.Selects = append(cloned.Selects,
			common.WithCountBySeverityAndFixabilityQuery(query, search.CVE).Selects...,
		)
	}

	return cloned
}
