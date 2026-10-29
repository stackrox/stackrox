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
	refreshedRec.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
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
	s.assertFreshness(priorRec.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
	s.assertFreshness(refreshedRec.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)
	s.assertFreshness(otherRec.GetId(), storage.ComplianceOperatorCheckResultV2_UNKNOWN)

	// Marking again is idempotent: the prior row is already STALE, nothing else changes.
	updated, err = s.dataStore.MarkStalePriorResultsByScanRef(s.hasWriteCtx, refreshedRec.GetScanRefId(), pivot)
	s.Require().NoError(err)
	s.Require().Zero(updated)
}

// TestMarkStalePriorDoesNotRevertConcurrentUpsert is a regression test for the lost-update race:
// the timeout hook's read-modify-upsert of the serialized proto must not overwrite a newer result
// written concurrently by the ingest path, and vice versa. The key fence serializes the two
// writers and makes the timeout hook re-read under the lock, so the newer Status survives.
func (s *complianceCheckResultDataStoreTestSuite) TestMarkStalePriorDoesNotRevertConcurrentUpsert() {
	now := time.Now()
	// pivot is the timed-out run's start time; the seeded rows predate it, so the timeout hook
	// wants to flip every row to STALE.
	pivot := now.Add(-5 * time.Minute)
	oldTime := now.Add(-10 * time.Hour)

	const n = 50
	recs := make([]*storage.ComplianceOperatorCheckResultV2, 0, n)
	var scanRefID string
	for i := 0; i < n; i++ {
		// All rows share one scan ref (same cluster + scan name) so a single
		// MarkStalePriorResultsByScanRef call targets the whole set.
		rec := getTestRecForScan("sc-race", testconsts.Cluster1, "scan-race")
		rec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
		rec.Status = storage.ComplianceOperatorCheckResultV2_INFO
		// getTestRecForScan derives the ID from the scan ref, so force distinct IDs per row.
		rec.Id = fmt.Sprintf("race-%d", i)
		s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec))
		recs = append(recs, rec)
		scanRefID = rec.GetScanRefId()
	}

	// Concurrently: the timeout hook flips every row UNKNOWN -> STALE, while the ingest path
	// rewrites each row with a new Status. Without the fence the hook could re-upsert its old
	// snapshot over the ingest, reverting Status to INFO. With the fence, the final Status must be
	// PASS.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := s.dataStore.MarkStalePriorResultsByScanRef(s.hasWriteCtx, scanRefID, pivot)
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

	// Every row keeps the ingest's newer Status (not reverted). The timeout hook's freshness
	// write either committed STALE or was superseded by the ingest write; in both cases the key
	// fence guarantees the Status is never rolled back to INFO.
	for _, rec := range recs {
		stored, found, err := s.storage.Get(s.hasReadCtx, rec.GetId())
		s.Require().NoError(err)
		s.Require().True(found)
		s.Equal(storage.ComplianceOperatorCheckResultV2_PASS, stored.GetStatus(), "Status reverted for %s", rec.GetId())
	}
}

func (s *complianceCheckResultDataStoreTestSuite) assertFreshness(id string, expected storage.ComplianceOperatorCheckResultV2_DataFreshness) {
	rec, found, err := s.storage.Get(s.hasReadCtx, id)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(expected, rec.GetDataFreshness())
}
