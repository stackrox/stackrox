package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	store "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

const (
	lastStartedTimestampColumnName = "laststartedtime"
	scanRefIDColumnName            = "scanrefid"
)

type datastoreImpl struct {
	store store.Store
	db    postgres.DB

	// keyFence prevents UpsertResult from racing MarkScanChecksStale.
	keyFence concurrency.KeyFence
}

// UpsertResult stores a result received from sensor. If enabling the use of this method from a
// service, the creation of the `ScanRefID` must be accounted for. In reality this method should only
// be used by the pipeline as this is a compliance operator object we are storing.
func (d *datastoreImpl) UpsertResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error {
	if ok, err := complianceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// TODO (ROX-20573): populate the standard and control from the rule so that lookup only happens
	// one time on insert and not everytime we pull the results.

	return d.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet([]byte(result.GetId())), func() error {
		if features.ComplianceSurfaceStaleData.Enabled() {
			existing, found, err := d.store.Get(ctx, result.GetId())
			if err != nil {
				return errors.Wrap(err, "looking up existing compliance check result to evaluate freshness")
			}
			reconcileReceivedResult(result, existing, found)
		}
		return d.store.Upsert(ctx, result)
	})
}

// reconcileReceivedResult sets incoming's DataFreshness to CURRENT only when it is newer than the
// stored row, otherwise preserves the stored freshness. incoming is mutated.
func reconcileReceivedResult(incoming, existing *storage.ComplianceOperatorCheckResultV2, found bool) {
	started := incoming.GetLastStartedTime()
	if !found {
		if started == nil {
			incoming.DataFreshness = storage.ComplianceOperatorCheckResultV2_UNKNOWN
		} else {
			incoming.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
		}
		return
	}

	// LastStartedTime comes from the operator's last-scanned annotation, so it advances only on a
	// real rerun. A strictly newer timestamp is the only genuine refresh.
	existingStarted := existing.GetLastStartedTime()
	if started != nil && (existingStarted == nil || started.AsTime().After(existingStarted.AsTime())) {
		incoming.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
		return
	}

	incoming.DataFreshness = existing.GetDataFreshness()
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
	var countResults []*ResourceResultCountByClusterScan
	err := pgSearch.RunSelectRequestForSchemaFn[ResourceResultCountByClusterScan](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery, func(r *ResourceResultCountByClusterScan) error {
		countResults = append(countResults, r)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return countResults, nil
}

// ComplianceProfileResultStats retrieves the profile result stats specified by query
func (d *datastoreImpl) ComplianceProfileResultStats(ctx context.Context, query *v1.Query) ([]*ResourceResultCountByProfile, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ComplianceOperatorCheckResultV2", "ComplianceProfileResultStats")

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
	var countResults []*ResourceResultCountByProfile
	err := pgSearch.RunSelectRequestForSchemaFn[ResourceResultCountByProfile](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery, func(r *ResourceResultCountByProfile) error {
		countResults = append(countResults, r)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return countResults, nil
}

// ComplianceProfileResults retrieves the profile results specified by query
func (d *datastoreImpl) ComplianceProfileResults(ctx context.Context, query *v1.Query) ([]*ResourceResultsByProfile, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ComplianceOperatorCheckResultV2", "ComplianceProfileResultStats")

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
	var results []*ResourceResultsByProfile
	err := pgSearch.RunSelectRequestForSchemaFn[ResourceResultsByProfile](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery, func(r *ResourceResultsByProfile) error {
		results = append(results, r)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// ComplianceClusterStats retrieves the scan result stats specified by query for the clusters
func (d *datastoreImpl) ComplianceClusterStats(ctx context.Context, query *v1.Query) ([]*ResultStatusCountByCluster, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ComplianceOperatorCheckResultV2", "ComplianceClusterStats")

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
	var countResults []*ResultStatusCountByCluster
	err := pgSearch.RunSelectRequestForSchemaFn[ResultStatusCountByCluster](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery, func(r *ResultStatusCountByCluster) error {
		countResults = append(countResults, r)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to retrieve data")
	}

	return countResults, nil
}

// CountByField retrieves the distinct scan result counts specified by query based on specified search field
func (d *datastoreImpl) CountByField(ctx context.Context, query *v1.Query, field search.FieldLabel) (int, error) {

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
	return pgSearch.RunDistinctCountForSchema(ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, query, search.ClusterID)
}

func (d *datastoreImpl) countByProfile(ctx context.Context, query *v1.Query) (int, error) {
	return pgSearch.RunDistinctCountForSchema(ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, query, search.ComplianceOperatorProfileName)
}

func (d *datastoreImpl) countByCheck(ctx context.Context, query *v1.Query) (int, error) {
	return pgSearch.RunDistinctCountForSchema(ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, query, search.ComplianceOperatorCheckName)
}

func (d *datastoreImpl) countByConfiguration(ctx context.Context, query *v1.Query) (int, error) {
	return pgSearch.RunDistinctCountForSchema(ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, query, search.ComplianceOperatorScanConfigName)
}

func (d *datastoreImpl) CountCheckResults(ctx context.Context, q *v1.Query) (int, error) {
	return d.store.Count(ctx, q)
}

func (d *datastoreImpl) DeleteResultsByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	return d.store.DeleteByQuery(ctx, query)
}

func (d *datastoreImpl) DeleteResultsByScanConfigAndCluster(ctx context.Context, scanConfigName string, clusterIDs []string) error {
	if scanConfigName == "" || len(clusterIDs) == 0 {
		return nil
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfigName, scanConfigName).AddExactMatches(search.ClusterID, clusterIDs...).ProtoQuery()
	return d.store.DeleteByQuery(ctx, query)
}

func (d *datastoreImpl) DeleteResultsByScans(ctx context.Context, scanRefIds []string) error {
	if len(scanRefIds) == 0 {
		return nil
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanRef, scanRefIds...).ProtoQuery()
	return d.store.DeleteByQuery(ctx, query)
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

func (d *datastoreImpl) DeleteOldResults(ctx context.Context, lastStartedTimestamp *timestamppb.Timestamp, scanRefID string, includeCurrent bool) error {
	if scanRefID == "" || lastStartedTimestamp == nil {
		return nil
	}
	if err := lastStartedTimestamp.CheckValid(); err != nil {
		return err
	}
	deleteFmt := "DELETE FROM %s WHERE %s = $1 AND (%s < $2 OR %s IS NULL)"
	if includeCurrent {
		deleteFmt = "DELETE FROM %s WHERE %s = $1 AND (%s <= $2 OR %s IS NULL)"
	}
	deleteStmt := fmt.Sprintf(deleteFmt,
		schema.ComplianceOperatorCheckResultV2TableName,
		scanRefIDColumnName,
		lastStartedTimestampColumnName,
		lastStartedTimestampColumnName,
	)
	_, err := d.db.Exec(ctx, deleteStmt, scanRefID, protocompat.NilOrTime(lastStartedTimestamp))
	return err
}

// freshnessUpdateBatchSize bounds how many changed results are re-serialized and upserted per
// UpsertMany call, so a large update does not issue one oversized write.
const freshnessUpdateBatchSize = 1000

// MarkScanChecksStale marks STALE the check results (from the scan specified via scanRefID) whose
// LastStartedTime predates olderThan.
func (d *datastoreImpl) MarkScanChecksStale(ctx context.Context, scanRefID string, olderThan time.Time) (int64, error) {
	if ok, err := complianceSAC.WriteAllowed(ctx); err != nil {
		return 0, err
	} else if !ok {
		return 0, sac.ErrResourceAccessDenied
	}
	if scanRefID == "" {
		return 0, nil
	}

	query := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanRef, scanRefID).
		ProtoQuery()

	return d.markStale(ctx, query, func(result *storage.ComplianceOperatorCheckResultV2) bool {
		started := result.GetLastStartedTime()
		return started != nil && started.AsTime().Before(olderThan)
	}, "marking prior compliance check results stale")
}

// markStale performs a read-modify-upsert that flips to STALE every result matched by query for
// which shouldStale reports true. Results already STALE, or for which shouldStale is false, are
// left untouched.
func (d *datastoreImpl) markStale(ctx context.Context, query *v1.Query, shouldStale func(*storage.ComplianceOperatorCheckResultV2) bool, errMsg string) (int64, error) {
	// Collect IDs during the read, upsert after it completes (upserting inside the callback would
	// need a second connection, risking a deadlock)
	var candidateIDs []string
	err := d.store.GetByQueryFn(ctx, query, func(result *storage.ComplianceOperatorCheckResultV2) error {
		if result.GetDataFreshness() != storage.ComplianceOperatorCheckResultV2_STALE && shouldStale(result) {
			candidateIDs = append(candidateIDs, result.GetId())
		}
		return nil
	})
	if err != nil {
		return 0, errors.Wrap(err, errMsg)
	}

	var updated int64
	for start := 0; start < len(candidateIDs); start += freshnessUpdateBatchSize {
		end := min(start+freshnessUpdateBatchSize, len(candidateIDs))
		batchIDs := candidateIDs[start:end]

		keys := make([][]byte, 0, len(batchIDs))
		for _, id := range batchIDs {
			keys = append(keys, []byte(id))
		}

		// Re-read and upsert under the key fence: freshness lives in the serialized proto, so the
		// whole row is rewritten. Ingest takes the same fence, so the re-read sees any concurrent
		// write and we never revert a newer result to an older snapshot.
		err := d.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keys...), func() error {
			results, _, err := d.store.GetMany(ctx, batchIDs)
			if err != nil {
				return err
			}
			changed := results[:0]
			for _, result := range results {
				if result.GetDataFreshness() == storage.ComplianceOperatorCheckResultV2_STALE || !shouldStale(result) {
					continue
				}
				result.DataFreshness = storage.ComplianceOperatorCheckResultV2_STALE
				changed = append(changed, result)
			}
			if len(changed) == 0 {
				return nil
			}
			if err := d.store.UpsertMany(ctx, changed); err != nil {
				return err
			}
			updated += int64(len(changed))
			return nil
		})
		if err != nil {
			return updated, errors.Wrap(err, errMsg)
		}
	}
	return updated, nil
}
