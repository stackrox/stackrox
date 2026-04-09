package deployments

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
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

// withTombstoneExclusion adds a filter to exclude soft-deleted (tombstoned) deployments,
// unless the caller has already explicitly referenced the TombstoneDeletedAt field.
// Pagination is stripped from the inner query and hoisted to the outer conjunction so
// that the SQL layer applies LIMIT/OFFSET from the top-level query as expected.
func withTombstoneExclusion(q *v1.Query) *v1.Query {
	if search.QueryMentionsField(q, search.TombstoneDeletedAt) {
		return q
	}
	excludeFilter := search.NewQueryBuilder().AddNullField(search.TombstoneDeletedAt).ProtoQuery()
	// Clone and strip pagination from the inner query before wrapping.
	inner := q.CloneVT()
	pagination := inner.GetPagination()
	inner.Pagination = nil
	outer := search.ConjunctionQuery(excludeFilter, inner)
	outer.Pagination = pagination
	return outer
}

func (v *deploymentViewImpl) Get(ctx context.Context, query *v1.Query) ([]DeploymentCore, error) {
	if err := common.ValidateQuery(query); err != nil {
		return nil, err
	}

	// Update the sort options to use aggregations if necessary as we are grouping by CVEs
	query = withTombstoneExclusion(query)
	query = common.UpdateSortAggs(query)
	query = withSelectQuery(query)

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]DeploymentCore, 0, paginated.GetLimit(query.GetPagination().GetLimit(), 100))
	err := pgSearch.RunSelectRequestForSchemaFn[deploymentResponse](queryCtx, v.db, v.schema, query, func(r *deploymentResponse) error {
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
			common.WithCountBySeverityAndFixabilityQuery(query, search.CVE).GetSelects()...,
		)
	}

	return cloned
}
