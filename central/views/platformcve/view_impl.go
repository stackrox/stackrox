package platformcve

import (
	"context"

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

type platformCVECoreViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *platformCVECoreViewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if err := common.ValidateQuery(q); err != nil {
		return 0, err
	}

	var err error
	q, err = common.WithSACFilter(ctx, resources.Cluster, q)
	if err != nil {
		return 0, err
	}

	var results []*platformCVECoreCount
	results, err = pgSearch.RunSelectRequestForSchema[platformCVECoreCount](ctx, v.db, v.schema, withCountQuery(q))
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

func (v *platformCVECoreViewImpl) Get(ctx context.Context, q *v1.Query) ([]CveCore, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	var err error
	q, err = common.WithSACFilter(ctx, resources.Cluster, q)
	if err != nil {
		return nil, err
	}

	var results []*platformCVECoreResponse
	results, err = pgSearch.RunSelectRequestForSchema[platformCVECoreResponse](ctx, v.db, v.schema, withSelectQuery(q))
	if err != nil {
		return nil, err
	}

	ret := make([]CveCore, 0, len(results))
	for _, r := range results {
		ret = append(ret, r)
	}
	return ret, nil
}

func (v *platformCVECoreViewImpl) GetClusterIDs(ctx context.Context, q *v1.Query) ([]string, error) {
	var err error
	q, err = common.WithSACFilter(ctx, resources.Cluster, q)
	if err != nil {
		return nil, err
	}

	q.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ClusterID).Proto(),
	}

	var results []*clusterResponse
	results, err = pgSearch.RunSelectRequestForSchema[clusterResponse](ctx, v.db, v.schema, q)
	if err != nil || len(results) == 0 {
		return nil, err
	}

	ret := make([]string, 0, len(results))
	for _, r := range results {
		ret = append(ret, r.ClusterID)
	}
	return ret, nil
}

func withSelectQuery(q *v1.Query) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.CVEID).Proto(),
		search.NewQuerySelect(search.CVEType).Proto(),
		search.NewQuerySelect(search.CVSS).Proto(),
		search.NewQuerySelect(search.CVECreatedTime).Proto(),
		search.NewQuerySelect(search.ClusterID).AggrFunc(aggregatefunc.Count).Proto(),
		search.NewQuerySelect(search.ClusterID).
			AggrFunc(aggregatefunc.Count).
			Filter("fixable_cluster_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
	}
	cloned.Selects = append(cloned.Selects, withCountByPlatformTypeSelectQuery(q).Selects...)
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVEID.String()},
	}
	return cloned
}

func withCountByPlatformTypeSelectQuery(q *v1.Query) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = append(cloned.Selects,
		search.NewQuerySelect(search.ClusterID).
			AggrFunc(aggregatefunc.Count).
			Filter("generic_cluster_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.PlatformType,
						storage.ClusterType_GENERIC_CLUSTER.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(search.ClusterID).
			AggrFunc(aggregatefunc.Count).
			Filter("kubernetes_cluster_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.PlatformType,
						storage.ClusterType_KUBERNETES_CLUSTER.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(search.ClusterID).
			AggrFunc(aggregatefunc.Count).
			Filter("openshift_cluster_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.PlatformType,
						storage.ClusterType_OPENSHIFT_CLUSTER.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(search.ClusterID).
			AggrFunc(aggregatefunc.Count).
			Filter("openshift4_cluster_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.PlatformType,
						storage.ClusterType_OPENSHIFT4_CLUSTER.String(),
					).ProtoQuery(),
			).Proto(),
	)
	return cloned
}

func withCountQuery(q *v1.Query) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVEID).AggrFunc(aggregatefunc.Count).Proto(),
	}
	return cloned
}
