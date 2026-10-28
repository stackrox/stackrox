//go:build sql_integration

package datastore

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sync"
)

func (s *complianceCheckResultDataStoreTestSuite) TestUpdateFreshnessByScanConfig() {
	now := time.Now()
	grace := 85 * time.Minute
	staleBefore := now.Add(-grace)

	// A result whose laststartedtime is older than staleBefore must become STALE; a recent one
	// must become CURRENT. Both start at the zero value (UNKNOWN), as a freshly inserted row does.
	oldTime := now.Add(-10 * time.Hour)
	recentTime := now.Add(-5 * time.Minute)
	staleRec := getTestRecForScan("sc-sweep", testconsts.Cluster1, "scan-stale")
	staleRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
	currentRec := getTestRecForScan("sc-sweep", testconsts.Cluster2, "scan-current")
	currentRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&recentTime)
	// A result whose last started time is nil cannot be evaluated and must stay UNKNOWN.
	nilRec := getTestRecForScan("sc-sweep", testconsts.Cluster1, "scan-nil")
	nilRec.LastStartedTime = nil
	// A result of a different scan config must be left untouched.
	otherRec := getTestRecForScan("sc-other", testconsts.Cluster1, "scan-other")
	otherRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)

	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, staleRec))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, currentRec))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, nilRec))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, otherRec))

	// Write context is required.
	_, err := s.dataStore.UpdateFreshnessByScanConfig(s.hasReadCtx, "sc-sweep", staleBefore)
	s.Require().Error(err)

	// First sweep flips the dated sc-sweep rows (UNKNOWN -> STALE / CURRENT); the nil-time row
	// stays UNKNOWN, so it is not counted among the updated rows.
	updated, err := s.dataStore.UpdateFreshnessByScanConfig(s.hasWriteCtx, "sc-sweep", staleBefore)
	s.Require().NoError(err)
	s.Require().Equal(int64(2), updated)

	s.assertFreshness(staleRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_STALE)
	s.assertFreshness(currentRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_CURRENT)
	// The nil-time row and the other scan config are untouched (still UNKNOWN).
	s.assertFreshness(nilRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_UNKNOWN)
	s.assertFreshness(otherRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_UNKNOWN)

	// Second sweep with no change writes zero rows (only rows whose freshness differs are upserted).
	updated, err = s.dataStore.UpdateFreshnessByScanConfig(s.hasWriteCtx, "sc-sweep", staleBefore)
	s.Require().NoError(err)
	s.Require().Zero(updated)

	// Moving the cutoff forward so the previously-current row is now stale flips exactly one row.
	updated, err = s.dataStore.UpdateFreshnessByScanConfig(s.hasWriteCtx, "sc-sweep", now.Add(1*time.Hour))
	s.Require().NoError(err)
	s.Require().Equal(int64(1), updated)
	s.assertFreshness(currentRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_STALE)

	// An empty scan config name is a no-op.
	updated, err = s.dataStore.UpdateFreshnessByScanConfig(s.hasWriteCtx, "", staleBefore)
	s.Require().NoError(err)
	s.Require().Zero(updated)
}

func (s *complianceCheckResultDataStoreTestSuite) TestMarkStalePriorResultsByScanRef() {
	now := time.Now()
	// pivot is the timed-out run's start time. A result the run refreshed carries this time; a
	// result it did not refresh carries an older time (a previous run) and is the stale leftover.
	pivot := now.Add(-5 * time.Minute)
	oldTime := now.Add(-10 * time.Hour)

	// Two results share a scan (same cluster + scan name => same scan ref): one was refreshed by
	// the timed-out run (recent), one was not (old). A third result belongs to a different scan
	// and must be left untouched.
	refreshedRec := getTestRecForScan("sc-timeout", testconsts.Cluster1, "scan-timed-out")
	refreshedRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&pivot)
	refreshedRec.DataFreshness = storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_CURRENT
	priorRec := getTestRecForScan("sc-timeout", testconsts.Cluster1, "scan-timed-out")
	priorRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
	otherRec := getTestRecForScan("sc-timeout", testconsts.Cluster2, "scan-healthy")
	otherRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
	s.Require().Equal(refreshedRec.GetScanRefId(), priorRec.GetScanRefId())
	s.Require().NotEqual(refreshedRec.GetScanRefId(), otherRec.GetScanRefId())

	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, refreshedRec))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, priorRec))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, otherRec))

	// Write context is required.
	_, err := s.dataStore.MarkStalePriorResultsByScanRef(s.hasReadCtx, refreshedRec.GetScanRefId(), pivot)
	s.Require().Error(err)

	// An empty scan ref is a no-op.
	updated, err := s.dataStore.MarkStalePriorResultsByScanRef(s.hasWriteCtx, "", pivot)
	s.Require().NoError(err)
	s.Require().Zero(updated)

	// The timeout marks only the prior (not-refreshed) row STALE. The row the run refreshed keeps
	// its CURRENT value; the other scan is untouched.
	updated, err = s.dataStore.MarkStalePriorResultsByScanRef(s.hasWriteCtx, refreshedRec.GetScanRefId(), pivot)
	s.Require().NoError(err)
	s.Require().Equal(int64(1), updated)
	s.assertFreshness(priorRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_STALE)
	s.assertFreshness(refreshedRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_CURRENT)
	s.assertFreshness(otherRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_UNKNOWN)

	// Marking again is idempotent: the prior row is already STALE, nothing else changes.
	updated, err = s.dataStore.MarkStalePriorResultsByScanRef(s.hasWriteCtx, refreshedRec.GetScanRefId(), pivot)
	s.Require().NoError(err)
	s.Require().Zero(updated)
}

// TestUpdateFreshnessDoesNotRevertConcurrentUpsert is a regression test for the lost-update
// race (PR #23605 review): the sweeper's read-modify-upsert of the serialized proto must not
// overwrite a newer result written concurrently by the ingest path. The key fence serializes
// the two writers and makes the sweeper re-read under the lock, so the newer Status survives.
func (s *complianceCheckResultDataStoreTestSuite) TestUpdateFreshnessDoesNotRevertConcurrentUpsert() {
	now := time.Now()
	staleBefore := now.Add(-85 * time.Minute)
	oldTime := now.Add(-10 * time.Hour)

	const n = 50
	recs := make([]*storage.ComplianceOperatorCheckResultV2, 0, n)
	for i := 0; i < n; i++ {
		rec := getTestRecForScan("sc-race", testconsts.Cluster1, fmt.Sprintf("scan-%d", i))
		rec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
		rec.Status = storage.ComplianceOperatorCheckResultV2_INFO
		s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec))
		recs = append(recs, rec)
	}

	// Concurrently: the sweeper flips every row UNKNOWN -> STALE, while the ingest path rewrites
	// each row with a new Status. Without the fence the sweeper could re-upsert its old snapshot
	// over the ingest, reverting Status to INFO. With the fence, the final Status must be PASS.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := s.dataStore.UpdateFreshnessByScanConfig(s.hasWriteCtx, "sc-race", staleBefore)
		s.NoError(err)
	}()
	go func() {
		defer wg.Done()
		for _, rec := range recs {
			updated := rec.CloneVT()
			updated.Status = storage.ComplianceOperatorCheckResultV2_PASS
			s.NoError(s.dataStore.UpsertResult(s.hasWriteCtx, updated))
		}
	}()
	wg.Wait()

	// Every row keeps the ingest's newer Status (not reverted), and the sweeper settled
	// freshness to STALE (its last read saw the old-enough LastStartedTime).
	for _, rec := range recs {
		stored, found, err := s.storage.Get(s.hasReadCtx, rec.GetId())
		s.Require().NoError(err)
		s.Require().True(found)
		s.Equal(storage.ComplianceOperatorCheckResultV2_PASS, stored.GetStatus(), "Status reverted for %s", rec.GetId())
	}
}

// TestTimeoutMarkingAndSweepAgree checks that the write-time timeout marking and the age-based
// sweeper produce the same result, so a later sweep does not revert the timeout marking. A
// timed-out run refreshes some results (recent start time) and leaves others from a prior run
// (old start time). The timeout marks only the prior ones STALE. The next sweep, using the same
// age cutoff, must keep that exact assignment.
func (s *complianceCheckResultDataStoreTestSuite) TestTimeoutMarkingAndSweepAgree() {
	now := time.Now()
	// The timed-out run started "now-ish"; its results are fresh.
	pivot := now.Add(-5 * time.Minute)
	// A prior run's result, older than one schedule period.
	oldTime := now.Add(-10 * time.Hour)
	// The sweep cutoff: expectedRefresh (~the timed-out run) minus the grace window.
	staleBefore := pivot.Add(-85 * time.Minute)

	refreshedRec := getTestRecForScan("sc-agree", testconsts.Cluster1, "scan-agree")
	refreshedRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&pivot)
	refreshedRec.DataFreshness = storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_CURRENT
	priorRec := getTestRecForScan("sc-agree", testconsts.Cluster1, "scan-agree")
	priorRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
	s.Require().Equal(refreshedRec.GetScanRefId(), priorRec.GetScanRefId())
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, refreshedRec))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, priorRec))

	// Timeout marks only the prior result STALE.
	_, err := s.dataStore.MarkStalePriorResultsByScanRef(s.hasWriteCtx, refreshedRec.GetScanRefId(), pivot)
	s.Require().NoError(err)
	s.assertFreshness(refreshedRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_CURRENT)
	s.assertFreshness(priorRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_STALE)

	// A later sweep must NOT revert: it writes zero rows and leaves both values as the timeout set
	// them (refreshed=CURRENT, prior=STALE).
	updated, err := s.dataStore.UpdateFreshnessByScanConfig(s.hasWriteCtx, "sc-agree", staleBefore)
	s.Require().NoError(err)
	s.Require().Zero(updated)
	s.assertFreshness(refreshedRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_CURRENT)
	s.assertFreshness(priorRec.GetId(), storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_STALE)
}

func (s *complianceCheckResultDataStoreTestSuite) assertFreshness(id string, expected storage.ComplianceDataFreshness) {
	rec, found, err := s.storage.Get(s.hasReadCtx, id)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(expected, rec.GetDataFreshness())
}
