package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	checkResultSearch "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/search"
	store "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	store    store.Store
	db       postgres.DB
	searcher checkResultSearch.Searcher
}

// UpsertResult adds the result to the database  If enabling the use of this
// method from a service, the creation of the `ScanRefID` must be accounted for.  In reality this
// method should only be used by the pipeline as this is a compliance operator object we are storing.
func (d *datastoreImpl) UpsertResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error {
	if ok, err := complianceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// TODO (ROX-20573): populate the standard and control from the rule so that lookup only happens
	// one time on insert and not everytime we pull the results.

	return d.store.Upsert(ctx, result)
}

// DeleteResult removes a result from the database
func (d *datastoreImpl) DeleteResult(ctx context.Context, id string) error {
	if ok, err := complianceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return d.store.Delete(ctx, id)
}

// SearchComplianceCheckResults retrieves the scan results specified by query
func (d *datastoreImpl) SearchComplianceCheckResults(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorCheckResultV2, error) {
	return d.store.GetByQuery(ctx, query)
}

// GetComplianceCheckResult returns the instance of the result specified by ID
func (d *datastoreImpl) GetComplianceCheckResult(ctx context.Context, complianceResultID string) (*storage.ComplianceOperatorCheckResultV2, bool, error) {
	return d.store.Get(ctx, complianceResultID)
}

// ComplianceCheckResultStats retrieves the scan results stats specified by query for the scan configuration
func (d *datastoreImpl) ComplianceCheckResultStats(ctx context.Context, query *v1.Query) ([]*ResourceResultCountByClusterScan, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ComplianceOperatorCheckResultV2", "ComplianceCheckResultStats")

	var err error
	query, err = withSACFilter(ctx, resources.Compliance, query)
	if err != nil {
		return nil, err
	}

	cloned := query.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ClusterID).Proto(),
		search.NewQuerySelect(search.Cluster).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorScanConfigName).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ClusterID.String(),
			search.Cluster.String(),
			search.ComplianceOperatorScanConfigName.String(),
		},
	}

	if cloned.Pagination == nil {
		cloned.Pagination = &v1.QueryPagination{}
	}
	cloned.Pagination.SortOptions = []*v1.QuerySortOption{
		{
			Field: search.ComplianceOperatorScanConfigName.String(),
		},
		{
			Field: search.ClusterID.String(),
		},
		{
			Field: search.Cluster.String(),
		},
	}

	countQuery := d.withCountByResultSelectQuery(cloned, search.ClusterID)
	countResults, err := pgSearch.RunSelectRequestForSchema[ResourceResultCountByClusterScan](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery)
	if err != nil {
		return nil, err
	}

	return countResults, nil
}

// ComplianceProfileResultStats retrieves the profile results stats specified by query for the scan configuration
func (d *datastoreImpl) ComplianceProfileResultStats(ctx context.Context, query *v1.Query) ([]*ResourceResultCountByProfile, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ComplianceOperatorCheckResultV2", "ComplianceProfileResultStats")

	var err error
	query, err = withSACFilter(ctx, resources.Compliance, query)
	if err != nil {
		return nil, err
	}

	cloned := query.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ComplianceOperatorProfileName).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ComplianceOperatorProfileName.String(),
		},
	}

	if cloned.Pagination == nil {
		cloned.Pagination = &v1.QueryPagination{}
	}
	cloned.Pagination.SortOptions = []*v1.QuerySortOption{
		{
			Field: search.ComplianceOperatorProfileName.String(),
		},
	}

	countQuery := d.withCountByResultSelectQuery(cloned, search.ComplianceOperatorProfileName)
	countResults, err := pgSearch.RunSelectRequestForSchema[ResourceResultCountByProfile](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery)
	if err != nil {
		return nil, err
	}

	return countResults, nil
}

// ComplianceClusterStats retrieves the scan result stats specified by query for the clusters
func (d *datastoreImpl) ComplianceClusterStats(ctx context.Context, query *v1.Query) ([]*ResultStatusCountByCluster, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ComplianceOperatorCheckResultV2", "ComplianceClusterStats")

	var err error
	query, err = withSACFilter(ctx, resources.Compliance, query)
	if err != nil {
		return nil, err
	}

	cloned := query.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ClusterID).Proto(),
		search.NewQuerySelect(search.Cluster).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ClusterID.String(),
			search.Cluster.String(),
		},
	}

	if cloned.Pagination == nil {
		cloned.Pagination = &v1.QueryPagination{}
		cloned.Pagination.SortOptions = []*v1.QuerySortOption{
			{
				Field: search.ClusterID.String(),
			},
			{
				Field: search.Cluster.String(),
			},
		}
	}

	countQuery := d.withCountByResultSelectQuery(cloned, search.ClusterID)
	countResults, err := pgSearch.RunSelectRequestForSchema[ResultStatusCountByCluster](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery)
	if err != nil {
		return nil, errors.Wrap(err, "unable to retrieve data")
	}

	return countResults, nil
}

// ComplianceClusterStatsCount retrieves the distinct scan result counts specified by query for the clusters
func (d *datastoreImpl) ComplianceClusterStatsCount(ctx context.Context, query *v1.Query) (int, error) {
	var err error
	query, err = withSACFilter(ctx, resources.Compliance, query)
	if err != nil {
		return 0, err
	}

	var results []*clusterStatsCount
	results, err = pgSearch.RunSelectRequestForSchema[clusterStatsCount](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, withCountQuery(query))
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	if len(results) > 1 {
		err = errors.Errorf("Retrieved multiple rows when only one row is expected for count query %q", query.String())
		utils.Should(err)
		return 0, err
	}
	return results[0].ClusterCount, nil
}

func (d *datastoreImpl) CountCheckResults(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}

func (d *datastoreImpl) DeleteResultsByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	_, err := d.store.DeleteByQuery(ctx, query)
	return err
}

func withCountQuery(q *v1.Query) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ClusterID).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
	}
	return cloned
}

func (d *datastoreImpl) withCountByResultSelectQuery(q *v1.Query, countOn search.FieldLabel) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = append(cloned.Selects,
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("pass_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_PASS.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("fail_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_FAIL.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("error_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_ERROR.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("info_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_INFO.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("manual_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_MANUAL.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("not_applicable_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_NOT_APPLICABLE.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("inconsistent_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_INCONSISTENT.String(),
					).ProtoQuery(),
			).Proto(),
	)
	return cloned
}

func withSACFilter(ctx context.Context, targetResource permissions.ResourceMetadata, query *v1.Query) (*v1.Query, error) {
	sacQueryFilter, err := pgSearch.GetReadSACQuery(ctx, targetResource)
	if err != nil {
		return nil, err
	}
	return search.FilterQueryByQuery(query, sacQueryFilter), nil
}
