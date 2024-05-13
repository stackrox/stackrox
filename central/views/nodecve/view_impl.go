package nodecve

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/utils"
)

type nodeCVECoreViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (n *nodeCVECoreViewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if err := common.ValidateQuery(q); err != nil {
		return 0, err
	}

	var err error
	q, err = common.WithSACFilter(ctx, resources.Node, q)
	if err != nil {
		return 0, err
	}

	var results []*nodeCVECoreCount
	results, err = pgSearch.RunSelectRequestForSchema[nodeCVECoreCount](ctx, n.db, n.schema, withCountQuery(q))
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	if len(results) > 1 {
		err = errors.Errorf("Retrieved multiple rows when only one row is expected for count query %q", q.String())
		utils.Should(err)
		return 0, err
	}
	return results[0].CVECount, nil
}
func (n *nodeCVECoreViewImpl) Get(ctx context.Context, q *v1.Query) ([]CveCore, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	var err error
	q, err = common.WithSACFilter(ctx, resources.Node, q)
	if err != nil {
		return nil, err
	}

	var results []*nodeCVECoreResponse
	results, err = pgSearch.RunSelectRequestForSchema[nodeCVECoreResponse](ctx, n.db, n.schema, withSelectQuery(q))
	if err != nil {
		return nil, err
	}

	ret := make([]CveCore, 0, len(results))
	for _, r := range results {
		// For each record, sort the IDs so that result looks consistent.
		sort.SliceStable(r.CVEIDs, func(i, j int) bool {
			return r.CVEIDs[i] < r.CVEIDs[j]
		})
		ret = append(ret, r)
	}
	return ret, nil
}

func (n *nodeCVECoreViewImpl) CountBySeverity(ctx context.Context, q *v1.Query) (ResourceCountByCVESeverity, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	var err error
	q, err = common.WithSACFilter(ctx, resources.Node, q)
	if err != nil {
		return nil, err
	}

	var results []*countByNodeCVESeverity
	results, err = pgSearch.RunSelectRequestForSchema[countByNodeCVESeverity](ctx, n.db, n.schema, withCountBySeveritySelectQuery(q, search.CVE))
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return NewEmptyResourceCountByCVESeverity(), nil
	}
	if len(results) > 1 {
		err = errors.Errorf("Retrieved multiple rows when only one row is expected for count query %q", q.String())
		utils.Should(err)
		return NewEmptyResourceCountByCVESeverity(), err
	}

	return &countByNodeCVESeverity{
		CriticalSeverityCount:  results[0].CriticalSeverityCount,
		ImportantSeverityCount: results[0].ImportantSeverityCount,
		ModerateSeverityCount:  results[0].ModerateSeverityCount,
		LowSeverityCount:       results[0].LowSeverityCount,
	}, nil
}

func (n *nodeCVECoreViewImpl) GetNodeIDs(ctx context.Context, q *v1.Query) ([]string, error) {
	var err error
	q, err = common.WithSACFilter(ctx, resources.Node, q)
	if err != nil {
		return nil, err
	}

	q.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.NodeID).Distinct().Proto(),
	}

	var results []*nodeResponse
	results, err = pgSearch.RunSelectRequestForSchema[nodeResponse](ctx, n.db, n.schema, q)
	if err != nil || len(results) == 0 {
		return nil, err
	}

	ret := make([]string, 0, len(results))
	for _, r := range results {
		ret = append(ret, r.NodeID)
	}
	return ret, nil
}

func withSelectQuery(q *v1.Query) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
		search.NewQuerySelect(search.CVSS).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.NodeID).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
		search.NewQuerySelect(search.CVECreatedTime).Proto(),
		search.NewQuerySelect(search.OperatingSystem).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
	}
	cloned.Selects = append(cloned.Selects, withCountBySeveritySelectQuery(q, search.NodeID).Selects...)

	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}
	return cloned
}

func withCountBySeveritySelectQuery(q *v1.Query, countOn search.FieldLabel) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = append(cloned.Selects,
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("critical_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("important_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("moderate_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("low_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
	)
	return cloned
}

func withCountQuery(q *v1.Query) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVEID).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
	}
	return cloned
}
