package nodecve

import (
	"context"
	"sort"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
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

	result, err := pgSearch.RunSelectOneForSchema[nodeCVECoreCount](ctx, n.db, n.schema, common.WithCountQuery(q, search.CVE))
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}
	return result.CVECount, nil
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

	ret := make([]CveCore, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	err = pgSearch.RunSelectRequestForSchemaFn[nodeCVECoreResponse](ctx, n.db, n.schema, withSelectQuery(q), func(r *nodeCVECoreResponse) error {
		// For each record, sort the IDs so that result looks consistent.
		sort.SliceStable(r.CVEIDs, func(i, j int) bool {
			return r.CVEIDs[i] < r.CVEIDs[j]
		})
		sort.SliceStable(r.NodeIDs, func(i, j int) bool {
			return r.NodeIDs[i] < r.NodeIDs[j]
		})
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (n *nodeCVECoreViewImpl) CountBySeverity(ctx context.Context, q *v1.Query) (common.ResourceCountByCVESeverity, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	var err error
	q, err = common.WithSACFilter(ctx, resources.Node, q)
	if err != nil {
		return nil, err
	}

	result, err := pgSearch.RunSelectOneForSchema[countByNodeCVESeverity](ctx, n.db, n.schema, common.WithCountBySeverityAndFixabilityQuery(q, search.CVE))
	if err != nil {
		return nil, err
	}
	if result == nil {
		return common.NewEmptyResourceCountByCVESeverity(), nil
	}
	return result, nil
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

	ret := make([]string, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	err = pgSearch.RunSelectRequestForSchemaFn[nodeResponse](ctx, n.db, n.schema, q, func(r *nodeResponse) error {
		ret = append(ret, r.GetNodeID())
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

func withSelectQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
		search.NewQuerySelect(search.CVSS).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.NodeID).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
		search.NewQuerySelect(search.CVECreatedTime).AggrFunc(aggregatefunc.Min).Proto(),
		search.NewQuerySelect(search.OperatingSystem).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
		search.NewQuerySelect(search.NodeID).Distinct().Proto(),
	}
	cloned.Selects = append(cloned.Selects, common.WithCountBySeverityAndFixabilityQuery(q, search.NodeID).Selects...)

	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}
	return cloned
}
