//go:build sql_integration

package datastore

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sync"
)

// TestUpsertResultDerivesFreshness covers how UpsertResult derives DataFreshness from the incoming
// result's LastStartedTime relative to the stored row, under the KeyFence shared with
// MarkScanChecksStale and with the freshness feature on. Seeds that set up a prior row go through
// s.storage directly so they are plain writes, independent of the derivation under test.
func (s *complianceCheckResultDataStoreTestSuite) TestUpsertResultDerivesFreshness() {
	s.T().Setenv(features.ComplianceSurfaceStaleData.EnvVar(), "true")
	now := time.Now()
	older := now.Add(-10 * time.Hour)
	newer := now.Add(-5 * time.Minute)

	// First sighting, non-nil LastStartedTime: CURRENT.
	firstSeen := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-first-seen")
	firstSeen.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&older)
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, firstSeen))
	s.assertFreshness(firstSeen.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)

	// First sighting, nil LastStartedTime: UNKNOWN, not CURRENT -- freshness cannot be evaluated
	// without a timestamp, and such a row stays UNKNOWN because MarkScanChecksStale skips
	// nil-timestamp rows.
	firstSeenNilTime := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-first-seen-nil")
	firstSeenNilTime.LastStartedTime = nil
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, firstSeenNilTime))
	s.assertFreshness(firstSeenNilTime.GetId(), storage.ComplianceOperatorCheckResultV2_UNKNOWN)

	// Newer than stored: CURRENT, regardless of the row's previous freshness (STALE here).
	seeded := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-newer")
	seeded.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&older)
	seeded.DataFreshness = storage.ComplianceOperatorCheckResultV2_STALE
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, seeded))
	refreshed := seeded.CloneVT()
	refreshed.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&newer)
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, refreshed))
	s.assertFreshness(seeded.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)

	// Same-timestamp replay: DataFreshness is carried over from the stored row, so a replay of the
	// same scan run does not reset a prior STALE marking. The rest of the content is still ingested.
	sameTimeReplayBase := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-replay-same")
	sameTimeReplayBase.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&newer)
	sameTimeReplayBase.DataFreshness = storage.ComplianceOperatorCheckResultV2_STALE
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, sameTimeReplayBase))
	sameTimeReplay := sameTimeReplayBase.CloneVT()
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, sameTimeReplay))
	s.assertFreshness(sameTimeReplayBase.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
	s.assertLastStartedTime(sameTimeReplayBase.GetId(), newer)

	// Older (out-of-order) replay: the stored row is newer, so the replay is ignored entirely. The
	// stored freshness and LastStartedTime are left untouched.
	oldReplayBase := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-replay-older")
	oldReplayBase.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&newer)
	oldReplayBase.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, oldReplayBase))
	oldReplay := oldReplayBase.CloneVT()
	oldReplay.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&older)
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, oldReplay))
	s.assertFreshness(oldReplayBase.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)
	s.assertLastStartedTime(oldReplayBase.GetId(), newer)

	// A replay with a missing timestamp (incoming nil) is treated as the oldest possible, so against
	// an existing row with a real timestamp it is ignored. The stored row keeps its timestamp.
	nilReplayBase := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-replay-nil")
	nilReplayBase.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&newer)
	nilReplayBase.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, nilReplayBase))
	nilReplay := nilReplayBase.CloneVT()
	nilReplay.LastStartedTime = nil
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, nilReplay))
	s.assertFreshness(nilReplayBase.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)
	s.assertLastStartedTime(nilReplayBase.GetId(), newer)

	// Write context is required.
	s.Require().Error(s.dataStore.UpsertResult(s.hasReadCtx, firstSeen))
}

func (s *complianceCheckResultDataStoreTestSuite) TestMarkScanChecksStale() {
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
	_, err := s.dataStore.MarkScanChecksStale(s.hasReadCtx, refreshedRec.GetScanRefId(), pivot)
	s.Require().Error(err)

	// An empty scan ref is a no-op.
	updated, err := s.dataStore.MarkScanChecksStale(s.hasWriteCtx, "", pivot)
	s.Require().NoError(err)
	s.Require().Zero(updated)

	// The timeout marks only the prior (not-refreshed) row STALE. The row the run refreshed keeps
	// its CURRENT value; the other scan is untouched.
	updated, err = s.dataStore.MarkScanChecksStale(s.hasWriteCtx, refreshedRec.GetScanRefId(), pivot)
	s.Require().NoError(err)
	s.Require().Equal(int64(1), updated)
	s.assertFreshness(priorRec.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
	s.assertFreshness(refreshedRec.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)
	s.assertFreshness(otherRec.GetId(), storage.ComplianceOperatorCheckResultV2_UNKNOWN)

	// Marking again is idempotent: the prior row is already STALE, nothing else changes.
	updated, err = s.dataStore.MarkScanChecksStale(s.hasWriteCtx, refreshedRec.GetScanRefId(), pivot)
	s.Require().NoError(err)
	s.Require().Zero(updated)
}

// TestMarkScanChecksStaleSubSecondBoundary guards the second-granularity trap in the DB pre-filter.
// MarkScanChecksStale pushes an upper bound on LastStartedTime to the DB, but the search DSL formats
// timestamps with second precision. A row whose LastStartedTime is in the same whole second as the
// pivot, but strictly before it, must still be marked STALE. It would be dropped if the pushed bound
// were the pivot floored to its second, so the bound is rounded up and the Go re-check decides. A row
// in that same second but strictly after the pivot must be left untouched, proving the Go re-check
// still trims the rows the widened bound over-selects.
func (s *complianceCheckResultDataStoreTestSuite) TestMarkScanChecksStaleSubSecondBoundary() {
	// Fixed instant with a sub-second component so flooring to the second is observable.
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	pivot := base.Add(700 * time.Millisecond)       // 12:00:00.700
	beforePivot := base.Add(300 * time.Millisecond) // 12:00:00.300, same second, before pivot
	afterPivot := base.Add(900 * time.Millisecond)  // 12:00:00.900, same second, after pivot

	// Both rows share one scan ref. The seed goes through the store directly so its freshness is not
	// re-derived; both start CURRENT, so the STALE-exclusion filter keeps them as candidates.
	beforeRec := getTestRecForScan("sc-subsecond", testconsts.Cluster1, "scan-subsecond")
	beforeRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&beforePivot)
	beforeRec.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
	afterRec := getTestRecForScan("sc-subsecond", testconsts.Cluster1, "scan-subsecond")
	afterRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&afterPivot)
	afterRec.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
	s.Require().Equal(beforeRec.GetScanRefId(), afterRec.GetScanRefId())

	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, beforeRec))
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, afterRec))

	updated, err := s.dataStore.MarkScanChecksStale(s.hasWriteCtx, beforeRec.GetScanRefId(), pivot)
	s.Require().NoError(err)
	s.Require().Equal(int64(1), updated)
	s.assertFreshness(beforeRec.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
	s.assertFreshness(afterRec.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)
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
	for i := range n {
		// All rows share one scan ref (same cluster + scan name) so a single
		// MarkScanChecksStale call targets the whole set.
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
	// PASS. Both goroutines are released from a shared start gate only once both are running, to
	// maximize the chance the race actually overlaps instead of passing by accident because one
	// side happened to finish before the other started.
	ready, start := s.newRaceGate(2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ready.Done()
		<-start
		_, err := s.dataStore.MarkScanChecksStale(s.hasWriteCtx, scanRefID, pivot)
		s.NoError(err)
	}()
	go func() {
		defer wg.Done()
		ready.Done()
		<-start
		for _, rec := range recs {
			updated := rec.CloneVT()
			updated.Status = storage.ComplianceOperatorCheckResultV2_PASS
			s.NoError(s.dataStore.UpsertResult(s.hasWriteCtx, updated))
		}
	}()
	ready.Wait()
	close(start)
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

// TestUpsertResultDoesNotRevertConcurrentTimeout is a regression test for the TOCTOU race found in
// review on #23765: the receive path used to read a result's existing freshness, decide
// CURRENT/carry-forward, and only then upsert -- all without holding the key fence across the read.
// A replayed result (same LastStartedTime as what is stored, so not genuinely new) decided from a
// stale read could overwrite a concurrent timeout's STALE marking with a freshness value computed
// before that marking happened. UpsertResult closes that gap by reading and writing under the same
// KeyFence used by MarkScanChecksStale. No matter which side's critical section runs
// first, the result converges to the objectively correct value: STALE, because the row's
// LastStartedTime genuinely predates the timed-out run's pivot. A replay that reads before the
// timeout commits carries CURRENT forward, a no-op immediately corrected by the timeout's own
// (serialized, not concurrent) read right after; a replay that reads after the timeout carries the
// already-STALE value forward.
func (s *complianceCheckResultDataStoreTestSuite) TestUpsertResultDoesNotRevertConcurrentTimeout() {
	s.T().Setenv(features.ComplianceSurfaceStaleData.EnvVar(), "true")
	now := time.Now()
	pivot := now.Add(-5 * time.Minute)
	oldTime := now.Add(-10 * time.Hour)

	const n = 50
	recs := make([]*storage.ComplianceOperatorCheckResultV2, 0, n)
	var scanRefID string
	for i := range n {
		// All rows share one scan ref so a single MarkScanChecksStale call targets the
		// whole set. They start CURRENT, as if a prior run had resolved them fresh; the pivot is
		// now newer than their LastStartedTime, so the timed-out run did not refresh them. Seed
		// through the store so the seed itself does not derive freshness.
		rec := getTestRecForScan("sc-replay-race", testconsts.Cluster1, "scan-replay-race")
		rec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
		rec.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
		rec.Id = fmt.Sprintf("replay-race-%d", i)
		s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, rec))
		recs = append(recs, rec)
		scanRefID = rec.GetScanRefId()
	}

	ready, start := s.newRaceGate(2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ready.Done()
		<-start
		_, err := s.dataStore.MarkScanChecksStale(s.hasWriteCtx, scanRefID, pivot)
		s.NoError(err)
	}()
	go func() {
		defer wg.Done()
		ready.Done()
		<-start
		for _, rec := range recs {
			// A replay: same ID, same LastStartedTime as what is stored. Not a genuinely new scan
			// run, so this must not force CURRENT regardless of how it interleaves with the
			// timeout.
			replayed := rec.CloneVT()
			s.NoError(s.dataStore.UpsertResult(s.hasWriteCtx, replayed))
		}
	}()
	ready.Wait()
	close(start)
	wg.Wait()

	for _, rec := range recs {
		s.assertFreshness(rec.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
	}
}

// newRaceGate returns a WaitGroup that n racing goroutines should call Done() on immediately after
// starting, and a channel they should block on before doing their real work. The caller waits on
// the group and then closes the channel, releasing all goroutines together so the race actually
// overlaps instead of passing by accident due to scheduling.
func (s *complianceCheckResultDataStoreTestSuite) newRaceGate(n int) (*sync.WaitGroup, chan struct{}) {
	ready := &sync.WaitGroup{}
	ready.Add(n)
	return ready, make(chan struct{})
}

func (s *complianceCheckResultDataStoreTestSuite) assertFreshness(id string, expected storage.ComplianceOperatorCheckResultV2_DataFreshness) {
	rec, found, err := s.storage.Get(s.hasReadCtx, id)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(expected, rec.GetDataFreshness())
}

func (s *complianceCheckResultDataStoreTestSuite) assertLastStartedTime(id string, expected time.Time) {
	rec, found, err := s.storage.Get(s.hasReadCtx, id)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().True(expected.Equal(rec.GetLastStartedTime().AsTime()), "expected %s, got %s", expected, rec.GetLastStartedTime().AsTime())
}
