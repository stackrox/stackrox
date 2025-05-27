package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	checkResultSearch "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/search"
	store "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	complianceUtils "github.com/stackrox/rox/central/complianceoperator/v2/utils"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
	query, err = complianceUtils.WithSACFilter(ctx, resources.Compliance, query)
	if err != nil {
		return nil, err
	}

	cloned := query.CloneVT()
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

	if cloned.GetPagination() == nil {
		cloned.Pagination = &v1.QueryPagination{}
	}
	if cloned.GetPagination().GetSortOptions() == nil {
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
	}

	countQuery := d.withCountByResultSelectQuery(cloned, search.ClusterID)
	countResults, err := pgSearch.RunSelectRequestForSchema[ResourceResultCountByClusterScan](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery)
	if err != nil {
		return nil, err
	}

	return countResults, nil
}

// ComplianceProfileResultStats retrieves the profile result stats specified by query
func (d *datastoreImpl) ComplianceProfileResultStats(ctx context.Context, query *v1.Query) ([]*ResourceResultCountByProfile, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ComplianceOperatorCheckResultV2", "ComplianceProfileResultStats")

	var err error
	query, err = complianceUtils.WithSACFilter(ctx, resources.Compliance, query)
	if err != nil {
		return nil, err
	}

	cloned := query.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ComplianceOperatorProfileName).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ComplianceOperatorProfileName.String(),
		},
	}

	if cloned.GetPagination() == nil {
		cloned.Pagination = &v1.QueryPagination{}
	}
	if cloned.GetPagination().GetSortOptions() == nil {
		cloned.Pagination.SortOptions = []*v1.QuerySortOption{
			{
				Field: search.ComplianceOperatorProfileName.String(),
			},
		}
	}

	countQuery := d.withCountByResultSelectQuery(cloned, search.ComplianceOperatorProfileName)
	countResults, err := pgSearch.RunSelectRequestForSchema[ResourceResultCountByProfile](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery)
	if err != nil {
		return nil, err
	}

	return countResults, nil
}

// ComplianceProfileResults retrieves the profile results specified by query
func (d *datastoreImpl) ComplianceProfileResults(ctx context.Context, query *v1.Query) ([]*ResourceResultsByProfile, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ComplianceOperatorCheckResultV2", "ComplianceProfileResultStats")

	var err error
	query, err = complianceUtils.WithSACFilter(ctx, resources.Compliance, query)
	if err != nil {
		return nil, err
	}

	cloned := query.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ComplianceOperatorProfileName).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorCheckName).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorCheckRationale).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorRuleName).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ComplianceOperatorProfileName.String(),
			search.ComplianceOperatorCheckName.String(),
			search.ComplianceOperatorCheckRationale.String(),
			search.ComplianceOperatorRuleName.String(),
		},
	}

	if cloned.GetPagination() == nil {
		cloned.Pagination = &v1.QueryPagination{}
	}
	if cloned.GetPagination().GetSortOptions() == nil {
		cloned.Pagination.SortOptions = []*v1.QuerySortOption{
			{
				Field: search.ComplianceOperatorProfileName.String(),
			},
			{
				Field: search.ComplianceOperatorCheckName.String(),
			},
			{
				Field: search.ComplianceOperatorCheckRationale.String(),
			},
			{
				Field: search.ComplianceOperatorRuleName.String(),
			},
		}
	}

	countQuery := d.withCountByResultSelectQuery(cloned, search.ComplianceOperatorProfileName)
	results, err := pgSearch.RunSelectRequestForSchema[ResourceResultsByProfile](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// ComplianceClusterStats retrieves the scan result stats specified by query for the clusters
func (d *datastoreImpl) ComplianceClusterStats(ctx context.Context, query *v1.Query) ([]*ResultStatusCountByCluster, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ComplianceOperatorCheckResultV2", "ComplianceClusterStats")

	var err error
	query, err = complianceUtils.WithSACFilter(ctx, resources.Compliance, query)
	if err != nil {
		return nil, err
	}

	cloned := query.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ClusterID).Proto(),
		search.NewQuerySelect(search.Cluster).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorScanLastExecutedTime).
			AggrFunc(aggregatefunc.Max).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ClusterID.String(),
			search.Cluster.String(),
		},
	}

	if cloned.GetPagination() == nil {
		cloned.Pagination = &v1.QueryPagination{}
	}
	if cloned.GetPagination().GetSortOptions() == nil {
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

// CountByField retrieves the distinct scan result counts specified by query based on specified search field
func (d *datastoreImpl) CountByField(ctx context.Context, query *v1.Query, field search.FieldLabel) (int, error) {
	var err error
	query, err = complianceUtils.WithSACFilter(ctx, resources.Compliance, query)
	if err != nil {
		return 0, err
	}

	switch field {
	case search.ClusterID:
		return d.countByCluster(ctx, query, field)
	case search.ComplianceOperatorProfileName:
		return d.countByProfile(ctx, query)
	case search.ComplianceOperatorCheckName:
		return d.countByCheck(ctx, query)
	case search.ComplianceOperatorScanConfigName:
		return d.countByConfiguration(ctx, query)
	}

	return 0, errors.Errorf("Unable to group result counts by %q", field)
}

func (d *datastoreImpl) WalkByQuery(ctx context.Context, query *v1.Query, fn func(result *storage.ComplianceOperatorCheckResultV2) error) error {
	wrappedFn := func(checkResult *storage.ComplianceOperatorCheckResultV2) error {
		return fn(checkResult)
	}
	return d.store.WalkByQuery(ctx, query, wrappedFn)
}

func (d *datastoreImpl) countByCluster(ctx context.Context, query *v1.Query, field search.FieldLabel) (int, error) {
	var results []*clusterCount
	results, err := pgSearch.RunSelectRequestForSchema[clusterCount](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, withCountQuery(query, search.ClusterID))
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
	return results[0].TotalCount, nil
}

func (d *datastoreImpl) countByProfile(ctx context.Context, query *v1.Query) (int, error) {
	var results []*profileCount
	results, err := pgSearch.RunSelectRequestForSchema[profileCount](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, withCountQuery(query, search.ComplianceOperatorProfileName))
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
	return results[0].TotalCount, nil
}

func (d *datastoreImpl) countByCheck(ctx context.Context, query *v1.Query) (int, error) {
	var results []*complianceCheckCount
	results, err := pgSearch.RunSelectRequestForSchema[complianceCheckCount](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, withCountQuery(query, search.ComplianceOperatorCheckName))
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
	return results[0].TotalCount, nil
}

func (d *datastoreImpl) countByConfiguration(ctx context.Context, query *v1.Query) (int, error) {
	var results []*configurationCount
	results, err := pgSearch.RunSelectRequestForSchema[configurationCount](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, withCountQuery(query, search.ComplianceOperatorScanConfigName))
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
	return results[0].TotalCount, nil
}

func (d *datastoreImpl) CountCheckResults(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}

func (d *datastoreImpl) DeleteResultsByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	_, err := d.store.DeleteByQuery(ctx, query)
	return err
}

func (d *datastoreImpl) DeleteResultsByScanConfigAndCluster(ctx context.Context, scanConfigName string, clusterIDs []string) error {
	if scanConfigName == "" || len(clusterIDs) == 0 {
		return nil
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfigName, scanConfigName).AddExactMatches(search.ClusterID, clusterIDs...).ProtoQuery()
	_, err := d.store.DeleteByQuery(ctx, query)

	return err
}

func (d *datastoreImpl) DeleteResultsByScans(ctx context.Context, scanRefIds []string) error {
	if len(scanRefIds) == 0 {
		return nil
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanRef, scanRefIds...).ProtoQuery()
	_, err := d.store.DeleteByQuery(ctx, query)

	return err
}

func withCountQuery(q *v1.Query, field search.FieldLabel) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(field).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
	}
	return cloned
}

func (d *datastoreImpl) withCountByResultSelectQuery(q *v1.Query, countOn search.FieldLabel) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = append(cloned.Selects,
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter(search.CompliancePassCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_PASS.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter(search.ComplianceFailCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_FAIL.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter(search.ComplianceErrorCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_ERROR.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter(search.ComplianceInfoCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_INFO.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter(search.ComplianceManualCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_MANUAL.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter(search.ComplianceNotApplicableCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_NOT_APPLICABLE.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter(search.ComplianceInconsistentCount.Alias(),
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_INCONSISTENT.String(),
					).ProtoQuery(),
			).Proto(),
	)
	return cloned
}
