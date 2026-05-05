package cveexport

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
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

var queryTimeout = env.PostgresVMStatementTimeout.DurationSetting()

type viewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *viewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if err := common.ValidateQuery(q); err != nil {
		return 0, err
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	result, err := pgSearch.RunSelectOneForSchema[cveCountResponse](queryCtx, v.db, v.schema, common.WithCountQuery(q, search.CVE))
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}
	return result.CVECount, nil
}

func (v *viewImpl) Get(ctx context.Context, q *v1.Query) ([]CveExport, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	cloned := q.CloneVT()
	cloned = common.UpdateSortAggs(cloned)

	var cveIDsToFilter []string
	var err error
	if cloned.GetPagination().GetLimit() > 0 || cloned.GetPagination().GetOffset() > 0 {
		cveIDsToFilter, err = v.getFilteredCVEs(ctx, cloned)
		if err != nil {
			return nil, err
		}
		if cloned.GetPagination() != nil && cloned.GetPagination().GetSortOptions() != nil {
			cloned.Pagination = &v1.QueryPagination{SortOptions: cloned.GetPagination().GetSortOptions()}
		}
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]CveExport, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	err = pgSearch.RunSelectRequestForSchemaFn[cveExportResponse](queryCtx, v.db, v.schema, withSelectCVEExportQuery(cloned, cveIDsToFilter), func(r *cveExportResponse) error {
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func withSelectCVEExportQuery(q *v1.Query, cveIDsToFilter []string) *v1.Query {
	cloned := q.CloneVT()
	if len(cveIDsToFilter) > 0 {
		cloned = search.ConjunctionQuery(cloned, search.NewQueryBuilder().AddDocIDs(cveIDsToFilter...).ProtoQuery())
		cloned.Pagination = q.GetPagination()
	}
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
		search.NewQuerySelect(search.Severity).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.CVSS).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.NVDCVSS).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.CVESummary).Proto(),
		search.NewQuerySelect(search.CVELink).Proto(),
		search.NewQuerySelect(search.CVEPublishedOn).AggrFunc(aggregatefunc.Min).Proto(),
		search.NewQuerySelect(search.EPSSProbablity).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.EPSSPercentile).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.AdvisoryName).Proto(),
		search.NewQuerySelect(search.AdvisoryLink).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}
	return cloned
}

func withSelectCVEIdentifiersQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}
	return cloned
}

func (v *viewImpl) getFilteredCVEs(ctx context.Context, q *v1.Query) ([]string, error) {
	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var cveIDsToFilter []string
	err := pgSearch.RunSelectRequestForSchemaFn[cveExportResponse](queryCtx, v.db, v.schema, withSelectCVEIdentifiersQuery(q), func(r *cveExportResponse) error {
		cveIDsToFilter = append(cveIDsToFilter, r.CVEIDs...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return cveIDsToFilter, nil
}
