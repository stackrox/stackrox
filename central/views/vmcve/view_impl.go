package vmcve

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
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
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

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]CveCore, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	err := pgSearch.RunSelectRequestForSchemaFn[vmCVECoreResponse](queryCtx, v.db, v.schema, cloned, func(r *vmCVECoreResponse) error {
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

func (v *vmCVECoreViewImpl) CountBySeverityPerVM(ctx context.Context, q *v1.Query) ([]VMSeverityCounts, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	cloned := common.WithCountBySeverityAndFixabilityQuery(q, search.VirtualMachineID)
	cloned.Selects = append([]*v1.QuerySelect{
		search.NewQuerySelect(search.VirtualMachineID).Proto(),
	}, cloned.GetSelects()...)
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.VirtualMachineID.String()},
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var ret []VMSeverityCounts
	err := pgSearch.RunSelectRequestForSchemaFn[vmSeverityCountsResponse](queryCtx, v.db, v.schema, cloned, func(r *vmSeverityCountsResponse) error {
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (v *vmCVECoreViewImpl) CountAffectedVMs(ctx context.Context, q *v1.Query) (int, error) {
	if err := common.ValidateQuery(q); err != nil {
		return 0, err
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	result, err := pgSearch.RunSelectOneForSchema[affectedVMCount](queryCtx, v.db, v.schema, common.WithCountQuery(q, search.VirtualMachineID))
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}
	return result.VMCount, nil
}

func (v *vmCVECoreViewImpl) GetAffectedVMs(ctx context.Context, q *v1.Query) ([]AffectedVMCore, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.VirtualMachineID).Proto(),
		search.NewQuerySelect(search.VirtualMachineName).Proto(),
		search.NewQuerySelect(search.Severity).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.Fixable).AggrFunc(aggregatefunc.Count).
			Filter("fixable_count",
				search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(search.CVSS).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.GuestOS).Proto(),
		search.NewQuerySelect(search.ComponentID).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.VirtualMachineID.String(),
			search.VirtualMachineName.String(),
			search.GuestOS.String(),
		},
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var ret []AffectedVMCore
	err := pgSearch.RunSelectRequestForSchemaFn[affectedVMResponse](queryCtx, v.db, v.schema, cloned, func(r *affectedVMResponse) error {
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
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
