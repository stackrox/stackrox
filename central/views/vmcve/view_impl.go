package vmcve

import (
	"context"
	"sort"

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

var (
	queryTimeout = env.PostgresVMStatementTimeout.DurationSetting()
)

type vmCVECoreViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *vmCVECoreViewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if err := common.ValidateQuery(q); err != nil {
		return 0, err
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	result, err := pgSearch.RunSelectOneForSchema[vmCVECoreCount](queryCtx, v.db, v.schema, common.WithCountQuery(q, search.CVE))
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}
	return result.CVECount, nil
}

func (v *vmCVECoreViewImpl) CountBySeverity(ctx context.Context, q *v1.Query) (common.ResourceCountByCVESeverity, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	result, err := pgSearch.RunSelectOneForSchema[resourceCountByVMCVESeverity](queryCtx, v.db, v.schema, common.WithCountBySeverityAndFixabilityQuery(q, search.VirtualMachineID))
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &resourceCountByVMCVESeverity{}, nil
	}
	return result, nil
}

func (v *vmCVECoreViewImpl) Get(ctx context.Context, q *v1.Query) ([]CveCore, error) {
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

	ret := make([]CveCore, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	err = pgSearch.RunSelectRequestForSchemaFn[vmCVECoreResponse](queryCtx, v.db, v.schema, withSelectCVECoreResponseQuery(cloned, cveIDsToFilter), func(r *vmCVECoreResponse) error {
		sort.SliceStable(r.CVEIDs, func(i, j int) bool {
			return r.CVEIDs[i] < r.CVEIDs[j]
		})
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (v *vmCVECoreViewImpl) GetVMIDs(ctx context.Context, q *v1.Query) ([]string, error) {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.VirtualMachineID).Distinct().Proto(),
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]string, 0, paginated.GetLimit(cloned.GetPagination().GetLimit(), 100))
	err := pgSearch.RunSelectRequestForSchemaFn[vmIDResponse](queryCtx, v.db, v.schema, cloned, func(r *vmIDResponse) error {
		ret = append(ret, r.VMID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	return ret, nil
}

func withSelectCVEIdentifiersQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}

	if common.IsSortBySeverityCounts(cloned) {
		cloned.Selects = append(cloned.Selects,
			common.WithCountBySeverityAndFixabilityQuery(q, search.VirtualMachineID).GetSelects()...,
		)
	}

	return cloned
}

func withSelectCVECoreResponseQuery(q *v1.Query, cveIDsToFilter []string) *v1.Query {
	cloned := q.CloneVT()
	if len(cveIDsToFilter) > 0 {
		cloned = search.ConjunctionQuery(cloned, search.NewQueryBuilder().AddDocIDs(cveIDsToFilter...).ProtoQuery())
		cloned.Pagination = q.GetPagination()
	}

	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
	}
	cloned.Selects = append(cloned.Selects,
		common.WithCountBySeverityAndFixabilityQuery(q, search.VirtualMachineID).GetSelects()...,
	)
	cloned.Selects = append(cloned.Selects,
		search.NewQuerySelect(search.CVSS).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.VirtualMachineID).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
		search.NewQuerySelect(search.CVECreatedTime).AggrFunc(aggregatefunc.Min).Proto(),
		search.NewQuerySelect(search.CVEPublishedOn).AggrFunc(aggregatefunc.Min).Proto(),
		search.NewQuerySelect(search.EPSSProbablity).AggrFunc(aggregatefunc.Max).Proto(),
	)
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}

	return cloned
}

func (v *vmCVECoreViewImpl) GetCVEComponents(ctx context.Context, q *v1.Query) ([]CVEComponentCore, error) {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.Component).Proto(),
		search.NewQuerySelect(search.ComponentVersion).Proto(),
		search.NewQuerySelect(search.ComponentSource).Proto(),
		search.NewQuerySelect(search.FixedBy).Proto(),
		search.NewQuerySelect(search.AdvisoryName).Proto(),
		search.NewQuerySelect(search.AdvisoryLink).Proto(),
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var ret []CVEComponentCore
	err := pgSearch.RunSelectRequestForSchemaFn[cveComponentResponse](queryCtx, v.db, v.schema, cloned, func(r *cveComponentResponse) error {
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (v *vmCVECoreViewImpl) getFilteredCVEs(ctx context.Context, q *v1.Query) ([]string, error) {
	var cveIDsToFilter []string

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	err := pgSearch.RunSelectRequestForSchemaFn[vmCVECoreResponse](queryCtx, v.db, v.schema, withSelectCVEIdentifiersQuery(q), func(r *vmCVECoreResponse) error {
		cveIDsToFilter = append(cveIDsToFilter, r.CVEIDs...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cveIDsToFilter, nil
}
