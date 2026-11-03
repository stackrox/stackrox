//go:build sql_integration

package datastore

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sync"
)

// TestUpsertResultDerivesFreshness covers how UpsertResult derives DataFreshness from the incoming
// result's LastStartedTime relative to the stored row, under the KeyFence shared with
// MarkScanChecksStale.
func (s *complianceCheckResultDataStoreTestSuite) TestUpsertResultDerivesFreshness() {
	s.T().Setenv(features.ComplianceSurfaceStaleData.EnvVar(), "true")
	now := time.Now()
	older := now.Add(-10 * time.Hour)
	newer := now.Add(-5 * time.Minute)

	firstSeen := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-first-seen")
	firstSeen.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&older)

	firstSeenNilTime := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-first-seen-nil")
	firstSeenNilTime.LastStartedTime = nil

	newerSeed := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-newer")
	newerSeed.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&older)
	newerSeed.DataFreshness = storage.ComplianceOperatorCheckResultV2_STALE
	newerIncoming := newerSeed.CloneVT()
	newerIncoming.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&newer)

	sameTimeSeed := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-replay-same")
	sameTimeSeed.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&newer)
	sameTimeSeed.DataFreshness = storage.ComplianceOperatorCheckResultV2_STALE
	sameTimeIncoming := sameTimeSeed.CloneVT()

	olderReplaySeed := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-replay-older")
	olderReplaySeed.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&newer)
	olderReplaySeed.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
	olderReplayIncoming := olderReplaySeed.CloneVT()
	olderReplayIncoming.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&older)

	nilReplaySeed := getTestRecForScan("sc-upsert-received", testconsts.Cluster1, "scan-replay-nil")
	nilReplaySeed.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&newer)
	nilReplaySeed.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
	nilReplayIncoming := nilReplaySeed.CloneVT()
	nilReplayIncoming.LastStartedTime = nil

	tests := map[string]struct {
		seed              *storage.ComplianceOperatorCheckResultV2
		incoming          *storage.ComplianceOperatorCheckResultV2
		expectFreshness   storage.ComplianceOperatorCheckResultV2_DataFreshness
		expectLastStarted *time.Time
	}{
		"first sighting, non-nil time: CURRENT": {
			incoming:        firstSeen,
			expectFreshness: storage.ComplianceOperatorCheckResultV2_CURRENT,
		},
		"first sighting, nil time: UNKNOWN": {
			incoming:        firstSeenNilTime,
			expectFreshness: storage.ComplianceOperatorCheckResultV2_UNKNOWN,
		},
		"newer than stored: CURRENT regardless of prior freshness": {
			seed:            newerSeed,
			incoming:        newerIncoming,
			expectFreshness: storage.ComplianceOperatorCheckResultV2_CURRENT,
		},
		"same-timestamp replay: freshness and LastStartedTime carried over": {
			seed:              sameTimeSeed,
			incoming:          sameTimeIncoming,
			expectFreshness:   storage.ComplianceOperatorCheckResultV2_STALE,
			expectLastStarted: &newer,
		},
		"older replay: ignored": {
			seed:              olderReplaySeed,
			incoming:          olderReplayIncoming,
			expectFreshness:   storage.ComplianceOperatorCheckResultV2_CURRENT,
			expectLastStarted: &newer,
		},
		"nil-incoming replay: ignored": {
			seed:              nilReplaySeed,
			incoming:          nilReplayIncoming,
			expectFreshness:   storage.ComplianceOperatorCheckResultV2_CURRENT,
			expectLastStarted: &newer,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			if tc.seed != nil {
				s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, tc.seed))
			}
			s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, tc.incoming))
			s.assertFreshness(tc.incoming.GetId(), tc.expectFreshness)
			if tc.expectLastStarted != nil {
				s.assertLastStartedTime(tc.incoming.GetId(), *tc.expectLastStarted)
			}
		})
	}

	s.Require().Error(s.dataStore.UpsertResult(s.hasReadCtx, firstSeen))
}

// TestMarkScanChecksStale covers MarkScanChecksStale, including the second-granularity trap in the
// DB pre-filter: the search DSL formats timestamps at second precision, so the pushed upper bound is
// rounded up to the next second and the Go re-check decides with full precision.
func (s *complianceCheckResultDataStoreTestSuite) TestMarkScanChecksStale() {
	now := time.Now()
	pivot := now.Add(-5 * time.Minute)
	oldTime := now.Add(-10 * time.Hour)

	s.Run("write context required", func() {
		_, err := s.dataStore.MarkScanChecksStale(s.hasReadCtx, "some-scan-ref", pivot)
		s.Require().Error(err)
	})

	s.Run("empty scan ref is a no-op", func() {
		updated, err := s.dataStore.MarkScanChecksStale(s.hasWriteCtx, "", pivot)
		s.Require().NoError(err)
		s.Require().Zero(updated)
	})

	s.Run("marks only the prior row stale, keeps refreshed row and other scan untouched", func() {
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

		updated, err := s.dataStore.MarkScanChecksStale(s.hasWriteCtx, refreshedRec.GetScanRefId(), pivot)
		s.Require().NoError(err)
		s.Require().Equal(int64(1), updated)
		s.assertFreshness(priorRec.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
		s.assertFreshness(refreshedRec.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)
		s.assertFreshness(otherRec.GetId(), storage.ComplianceOperatorCheckResultV2_UNKNOWN)
	})

	s.Run("second call is idempotent", func() {
		refreshedRec := getTestRecForScan("sc-timeout-idem", testconsts.Cluster1, "scan-timed-out")
		refreshedRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&pivot)
		priorRec := getTestRecForScan("sc-timeout-idem", testconsts.Cluster1, "scan-timed-out")
		priorRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
		s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, refreshedRec))
		s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, priorRec))

		updated, err := s.dataStore.MarkScanChecksStale(s.hasWriteCtx, refreshedRec.GetScanRefId(), pivot)
		s.Require().NoError(err)
		s.Require().Equal(int64(1), updated)

		updated, err = s.dataStore.MarkScanChecksStale(s.hasWriteCtx, refreshedRec.GetScanRefId(), pivot)
		s.Require().NoError(err)
		s.Require().Zero(updated)
	})

	s.Run("sub-second boundary: before pivot in the same second is STALE, after is CURRENT", func() {
		base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		subPivot := base.Add(700 * time.Millisecond)    // 12:00:00.700
		beforePivot := base.Add(300 * time.Millisecond) // 12:00:00.300, same second, before pivot
		afterPivot := base.Add(900 * time.Millisecond)  // 12:00:00.900, same second, after pivot

		beforeRec := getTestRecForScan("sc-subsecond", testconsts.Cluster1, "scan-subsecond")
		beforeRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&beforePivot)
		beforeRec.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
		afterRec := getTestRecForScan("sc-subsecond", testconsts.Cluster1, "scan-subsecond")
		afterRec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&afterPivot)
		afterRec.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
		s.Require().Equal(beforeRec.GetScanRefId(), afterRec.GetScanRefId())

		s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, beforeRec))
		s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, afterRec))

		updated, err := s.dataStore.MarkScanChecksStale(s.hasWriteCtx, beforeRec.GetScanRefId(), subPivot)
		s.Require().NoError(err)
		s.Require().Equal(int64(1), updated)
		s.assertFreshness(beforeRec.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
		s.assertFreshness(afterRec.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)
	})
}

// TestConcurrentUpsertVsStaleMarking is a regression test for the lost-update race between the
// timeout hook (MarkScanChecksStale) and the ingest path (UpsertResult): both read-modify-upsert the
// serialized proto for the same row, so without a shared key fence one writer can overwrite the
// other's result with a stale snapshot. The fence makes each side re-read under the lock, so the
// outcome below is deterministic regardless of which goroutine's critical section runs first.
func (s *complianceCheckResultDataStoreTestSuite) TestConcurrentUpsertVsStaleMarking() {
	s.T().Setenv(features.ComplianceSurfaceStaleData.EnvVar(), "true")
	now := time.Now()
	pivot := now.Add(-5 * time.Minute)
	oldTime := now.Add(-10 * time.Hour)
	evenOlder := now.Add(-20 * time.Hour)

	const n = 50

	tests := map[string]struct {
		scanName            string
		incomingLastStarted time.Time
		expectStatus        storage.ComplianceOperatorCheckResultV2_CheckStatus
		expectFreshness     storage.ComplianceOperatorCheckResultV2_DataFreshness
	}{
		"replay (same timestamp)": {
			scanName:            "scan-race-replay",
			incomingLastStarted: oldTime,
			expectStatus:        storage.ComplianceOperatorCheckResultV2_PASS,
			expectFreshness:     storage.ComplianceOperatorCheckResultV2_STALE,
		},
		"older upsert": {
			scanName:            "scan-race-older",
			incomingLastStarted: evenOlder,
			expectStatus:        storage.ComplianceOperatorCheckResultV2_INFO,
			expectFreshness:     storage.ComplianceOperatorCheckResultV2_STALE,
		},
		"newer upsert": {
			scanName:            "scan-race-newer",
			incomingLastStarted: now,
			expectStatus:        storage.ComplianceOperatorCheckResultV2_PASS,
			expectFreshness:     storage.ComplianceOperatorCheckResultV2_CURRENT,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			recs := make([]*storage.ComplianceOperatorCheckResultV2, 0, n)
			var scanRefID string
			for i := range n {
				rec := getTestRecForScan("sc-race", testconsts.Cluster1, tc.scanName)
				rec.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
				rec.Status = storage.ComplianceOperatorCheckResultV2_INFO
				rec.DataFreshness = storage.ComplianceOperatorCheckResultV2_CURRENT
				rec.Id = fmt.Sprintf("race-%s-%d", tc.scanName, i)
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
					incoming := rec.CloneVT()
					incoming.Status = storage.ComplianceOperatorCheckResultV2_PASS
					incoming.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&tc.incomingLastStarted)
					s.NoError(s.dataStore.UpsertResult(s.hasWriteCtx, incoming))
				}
			}()
			ready.Wait()
			close(start)
			wg.Wait()

			for _, rec := range recs {
				stored, found, err := s.storage.Get(s.hasReadCtx, rec.GetId())
				s.Require().NoError(err)
				s.Require().True(found)
				s.Equal(tc.expectStatus, stored.GetStatus(), "Status mismatch for %s", rec.GetId())
				s.Equal(tc.expectFreshness, stored.GetDataFreshness(), "DataFreshness mismatch for %s", rec.GetId())
			}
		})
	}
}

// TestMarkScanChecksStaleByConfig covers the time-driven sweeper's datastore method: keyed by scan
// configuration name (not scan ref), it marks STALE only the matched rows whose LastStartedTime
// predates the cutoff, leaving recent rows, nil-timestamp rows, and other configurations untouched.
func (s *complianceCheckResultDataStoreTestSuite) TestMarkScanChecksStaleByConfig() {
	now := time.Now()
	// cutoff sits between the old results (which predate it) and the recent one (which does not).
	cutoff := now.Add(-1 * time.Hour)
	oldTime := now.Add(-10 * time.Hour)
	recentTime := now.Add(-5 * time.Minute)

	const cfgA = "cfgA"
	const cfgB = "cfgB"

	// cfgA spans two clusters with mixed LastStartedTime: old (should go STALE), recent (newer than
	// the cutoff, must stay), and nil (freshness cannot be evaluated, must stay).
	oldA1 := getTestRecForScan(cfgA, testconsts.Cluster1, "scan-a-old")
	oldA1.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
	oldA2 := getTestRecForScan(cfgA, testconsts.Cluster2, "scan-a-old")
	oldA2.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
	recentA := getTestRecForScan(cfgA, testconsts.Cluster1, "scan-a-recent")
	recentA.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&recentTime)
	nilA := getTestRecForScan(cfgA, testconsts.Cluster2, "scan-a-nil")
	nilA.LastStartedTime = nil

	// cfgB predates the cutoff too, but a different scan configuration must never be touched.
	oldB := getTestRecForScan(cfgB, testconsts.Cluster1, "scan-b-old")
	oldB.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)

	for _, rec := range []*storage.ComplianceOperatorCheckResultV2{oldA1, oldA2, recentA, nilA, oldB} {
		s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec))
	}

	// Write context is required.
	_, err := s.dataStore.MarkScanChecksStaleByConfig(s.hasReadCtx, cfgA, cutoff)
	s.Require().Error(err)

	// An empty config name is a no-op.
	updated, err := s.dataStore.MarkScanChecksStaleByConfig(s.hasWriteCtx, "", cutoff)
	s.Require().NoError(err)
	s.Require().Zero(updated)

	// Only the two old cfgA rows predate the cutoff, so only they are marked STALE.
	updated, err = s.dataStore.MarkScanChecksStaleByConfig(s.hasWriteCtx, cfgA, cutoff)
	s.Require().NoError(err)
	s.Require().Equal(int64(2), updated)
	s.assertFreshness(oldA1.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
	s.assertFreshness(oldA2.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
	// Recent row (newer than cutoff) and nil-timestamp row keep their initial UNKNOWN.
	s.assertFreshness(recentA.GetId(), storage.ComplianceOperatorCheckResultV2_UNKNOWN)
	s.assertFreshness(nilA.GetId(), storage.ComplianceOperatorCheckResultV2_UNKNOWN)
	// The other scan configuration is untouched even though its row predates the cutoff.
	s.assertFreshness(oldB.GetId(), storage.ComplianceOperatorCheckResultV2_UNKNOWN)

	// Marking again is idempotent: the two rows are already STALE, nothing else changes.
	updated, err = s.dataStore.MarkScanChecksStaleByConfig(s.hasWriteCtx, cfgA, cutoff)
	s.Require().NoError(err)
	s.Require().Zero(updated)
}

// TestMarkScanChecksStaleByConfigAgreesWithWriteHooks verifies the time-driven sweeper never
// disagrees with the write-time hooks: with a sweeper-style cutoff (expected refresh minus the two
// scan-watcher timeouts), a row the ingest hook made CURRENT with a recent timestamp is not
// reverted, and a row the timeout hook already made STALE stays STALE. The net effect is zero
// changes, and re-running is a no-op.
func (s *complianceCheckResultDataStoreTestSuite) TestMarkScanChecksStaleByConfigAgreesWithWriteHooks() {
	s.T().Setenv(features.ComplianceSurfaceStaleData.EnvVar(), "true")
	now := time.Now()
	// grace mirrors the freshness sweeper's: the sum of the two scan-watcher timeouts.
	grace := env.ComplianceScanScheduleWatcherTimeout.DurationSetting() + env.ComplianceScanWatcherTimeout.DurationSetting()
	// Sweeper-style cutoff: a result refreshed within one grace window of the expected refresh
	// (taken as now here) is fresh and must not be swept.
	cutoff := now.Add(-grace)
	recentTime := now.Add(-10 * time.Minute) // after the cutoff: genuinely fresh
	oldTime := now.Add(-10 * time.Hour)      // before the cutoff
	pivot := now.Add(-5 * time.Minute)

	const cfg = "cfg-agree"

	// Fresh rows: ingested with a recent LastStartedTime, so the write-time hook derives CURRENT.
	fresh1 := getTestRecForScan(cfg, testconsts.Cluster1, "scan-fresh-1")
	fresh1.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&recentTime)
	fresh2 := getTestRecForScan(cfg, testconsts.Cluster2, "scan-fresh-2")
	fresh2.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&recentTime)
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, fresh1))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, fresh2))
	s.assertFreshness(fresh1.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)
	s.assertFreshness(fresh2.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)

	// Stale rows: share one scan ref, ingested CURRENT, then marked STALE by the timeout hook.
	staleA := getTestRecForScan(cfg, testconsts.Cluster3, "scan-stale")
	staleA.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
	staleB := getTestRecForScan(cfg, testconsts.Cluster3, "scan-stale")
	staleB.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&oldTime)
	s.Require().Equal(staleA.GetScanRefId(), staleB.GetScanRefId())
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, staleA))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, staleB))
	marked, err := s.dataStore.MarkScanChecksStale(s.hasWriteCtx, staleA.GetScanRefId(), pivot)
	s.Require().NoError(err)
	s.Require().Equal(int64(2), marked)
	s.assertFreshness(staleA.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
	s.assertFreshness(staleB.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)

	// The sweeper agrees with both hooks: fresh rows stay CURRENT (newer than the cutoff), already
	// STALE rows stay STALE, so nothing changes.
	updated, err := s.dataStore.MarkScanChecksStaleByConfig(s.hasWriteCtx, cfg, cutoff)
	s.Require().NoError(err)
	s.Require().Zero(updated)
	s.assertFreshness(fresh1.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)
	s.assertFreshness(fresh2.GetId(), storage.ComplianceOperatorCheckResultV2_CURRENT)
	s.assertFreshness(staleA.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)
	s.assertFreshness(staleB.GetId(), storage.ComplianceOperatorCheckResultV2_STALE)

	// Re-running is a no-op.
	updated, err = s.dataStore.MarkScanChecksStaleByConfig(s.hasWriteCtx, cfg, cutoff)
	s.Require().NoError(err)
	s.Require().Zero(updated)
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
