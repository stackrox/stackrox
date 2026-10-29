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

	// keyFence serializes the per-row ingest write (UpsertResult) against the read-modify-upsert
	// of a check result's serialized proto in MarkStalePriorResultsByScanRef. Freshness lives
	// inside the serialized blob, so the timeout hook loads, mutates, and re-upserts the whole
	// proto. Without the fence a concurrent ingest writing a newer result between the load and the
	// re-upsert would be reverted (lost update), and vice versa. Mirrors the CVE suppress
	// precedent (central/cve/node/datastore), which guards the same pattern with a KeyFence.
	keyFence concurrency.KeyFence
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

	// Hold the key fence for this result so a concurrent freshness read-modify-upsert
	// (MarkStalePriorResultsByScanRef) cannot re-upsert an older snapshot of this row over the
	// result written here.
	return d.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet([]byte(result.GetId())), func() error {
		return d.store.Upsert(ctx, result)
	})
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

// MarkStalePriorResultsByScanRef marks STALE the check results of the given scan (scan ref ID)
// that predate olderThan, leaving the rest unchanged. It is the write-time staleness hook used
// when a scan times out: the results that the timed-out run already refreshed carry its start time
// (olderThan) and are genuinely fresh, so they stay as they are. Only the results the run did NOT
// refresh (leftovers from a previous run, with an older last started time) are stale, and marking
// them here surfaces that immediately. A result row is reused across runs (its ID is the
// compliance operator check result UID), so "prior results" are the rows the latest run did not
// overwrite. Results whose last started time is nil are left unchanged (their freshness cannot be
// evaluated). Returns the number of rows updated.
func (d *datastoreImpl) MarkStalePriorResultsByScanRef(ctx context.Context, scanRefID string, olderThan time.Time) (int64, error) {
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

	return d.updateFreshness(ctx, query, func(result *storage.ComplianceOperatorCheckResultV2) storage.ComplianceOperatorCheckResultV2_DataFreshness {
		started := result.GetLastStartedTime()
		if started != nil && started.AsTime().Before(olderThan) {
			return storage.ComplianceOperatorCheckResultV2_STALE
		}
		// A result refreshed by the timed-out run (or with no start time) is left as-is.
		return result.GetDataFreshness()
	}, "marking prior compliance check results stale")
}

// updateFreshness performs a read-modify-upsert of data_freshness for the results matched by
// query. want returns the freshness a result should have.
//
// Two concerns shape the flow:
//
//  1. Deadlock avoidance: the candidate IDs are collected DURING the streaming read, and the
//     re-read + upsert happen AFTER the read completes. Upserting inside the GetByQueryFn callback
//     would issue writes on a second pool connection while the read stream still holds the first
//     one open; under a large result set Postgres blocks writing to the client (full send buffer)
//     while the client blocks acquiring a connection for the upsert, a deadlock. Only candidate
//     IDs are retained, so memory stays proportional to churn, not to the whole result set.
//
//  2. Lost-update avoidance: freshness lives inside the serialized proto, so the whole row is
//     re-upserted. A concurrent ingest (UpsertResult) could write a newer result between the read
//     and the upsert; re-upserting the old snapshot would revert it. Each batch therefore takes
//     the key fence on its IDs, re-reads the current rows under the lock, recomputes freshness,
//     and upserts only rows that still need it. Ingest takes the same fence, so inside the lock
//     the re-read always sees the latest row. This matches the CVE suppress precedent, which
//     guards its read-modify-upsert with a KeyFence.
func (d *datastoreImpl) updateFreshness(ctx context.Context, query *v1.Query, want func(*storage.ComplianceOperatorCheckResultV2) storage.ComplianceOperatorCheckResultV2_DataFreshness, errMsg string) (int64, error) {
	var candidateIDs []string
	err := d.store.GetByQueryFn(ctx, query, func(result *storage.ComplianceOperatorCheckResultV2) error {
		if result.GetDataFreshness() != want(result) {
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

		err := d.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keys...), func() error {
			results, _, err := d.store.GetMany(ctx, batchIDs)
			if err != nil {
				return err
			}
			changed := results[:0]
			for _, result := range results {
				w := want(result)
				if result.GetDataFreshness() == w {
					continue
				}
				result.DataFreshness = w
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
