package relay

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	relaytest "github.com/stackrox/rox/compliance/virtualmachines/relay/testutils"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
	"golang.org/x/time/rate"
)

func TestRelay(t *testing.T) {
	suite.Run(t, new(relayTestSuite))
}

type relayTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *relayTestSuite) SetupTest() {
	s.ctx = context.Background()
}

// errTestStreamStart is returned by the mock stream's Start method to simulate
// a startup failure.
var errTestStreamStart = errors.New("test stream start failure")

// errTest is a generic test error for use in mock implementations.
var errTest = errors.New("test error")

// TestRelay_StartFailure verifies that Relay.Run propagates stream startup
// errors and does not enter the main select loop when initialization fails.
func (s *relayTestSuite) TestRelay_StartFailure() {
	// Use a bounded context to ensure the test fails if Relay.Run blocks.
	ctx, cancel := context.WithTimeout(s.ctx, 100*time.Millisecond)
	defer cancel()

	// Create a stream that fails immediately on Start.
	stream := &failingIndexReportStream{}

	// Create a dummy sender. It should never be used in this test because the
	// relay is expected to fail before entering its main loop.
	sender := &mockIndexReportSender{
		failOnIndex:   -1,
		expectedCount: 0,
	}

	// Construct the relay under test.
	relay := New(stream, sender, &mockUMH{}, 0, 4*time.Hour, 0, 0)

	// Run the relay in a goroutine so we can assert it returns promptly and
	// does not block in its select loop.
	errCh := make(chan error, 1)
	go func() {
		errCh <- relay.Run(ctx)
	}()

	select {
	case err := <-errCh:
		// Relay.Run should surface the stream startup error (possibly wrapped).
		s.Require().Error(err, "Relay.Run should return an error when stream Start fails")
		s.Require().ErrorIs(err, errTestStreamStart, "Relay.Run should wrap the stream startup error")
		s.Equal(1, stream.startCalled, "stream.Start should be called exactly once")
	case <-time.After(100 * time.Millisecond):
		s.Fail("Relay.Run did not return promptly on stream Start failure (likely entered select loop)")
	}
}

// TestRelay_Integration tests the interaction between stream, relay, and sender.
// This uses mock implementations to verify the full data flow without real vsock/sensor.
func (s *relayTestSuite) TestRelay_Integration() {
	// Create mock sender that signals when reports are received
	done := concurrency.NewSignal()
	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex:   -1, // never fail
		done:          &done,
		expectedCount: 2,
	}

	// Create mock stream that produces test messages with discovered data
	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			{
				IndexReport: &v1.IndexReport{VsockCid: "100"},
				DiscoveredData: &v1.DiscoveredData{
					DetectedOs:        v1.DetectedOS_RHEL,
					OsVersion:         "9.3",
					ActivationStatus:  v1.ActivationStatus_ACTIVE,
					DnfMetadataStatus: v1.DnfMetadataStatus_AVAILABLE,
				},
			},
			{
				IndexReport: &v1.IndexReport{VsockCid: "200"},
				DiscoveredData: &v1.DiscoveredData{
					DetectedOs:        v1.DetectedOS_UNKNOWN,
					OsVersion:         "unknown",
					ActivationStatus:  v1.ActivationStatus_INACTIVE,
					DnfMetadataStatus: v1.DnfMetadataStatus_UNAVAILABLE,
				},
			},
		},
	}

	// Create relay with mock dependencies using the public constructor
	// Rate limiting disabled (0)
	relay := New(mockIndexReportStream, mockIndexReportSender, &mockUMH{}, 0, 4*time.Hour, 0, 0)

	// Run relay in background
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for all reports to be processed (or timeout)
	select {
	case <-done.Done():
		// All reports processed
	case <-time.After(1 * time.Second):
		s.Fail("Timeout waiting for reports to be processed")
	}

	cancel()

	// Verify all messages were sent with discovered data preserved
	var sentMessages []*v1.VMReport
	concurrency.WithLock(&mockIndexReportSender.mu, func() {
		sentMessages = append(sentMessages, mockIndexReportSender.sentMessages...)
	})
	s.Require().Len(sentMessages, 2)

	first := sentMessages[0]
	second := sentMessages[1]

	// IndexReport fields are preserved
	s.Equal("100", first.GetIndexReport().GetVsockCid())
	s.Equal("200", second.GetIndexReport().GetVsockCid())

	// VM discovered data is preserved
	s.Equal(v1.DetectedOS_RHEL, first.GetDiscoveredData().GetDetectedOs())
	s.Equal("9.3", first.GetDiscoveredData().GetOsVersion())
	s.Equal(v1.ActivationStatus_ACTIVE, first.GetDiscoveredData().GetActivationStatus())
	s.Equal(v1.DnfMetadataStatus_AVAILABLE, first.GetDiscoveredData().GetDnfMetadataStatus())

	s.Equal(v1.DetectedOS_UNKNOWN, second.GetDiscoveredData().GetDetectedOs())
	s.Equal("unknown", second.GetDiscoveredData().GetOsVersion())
	s.Equal(v1.ActivationStatus_INACTIVE, second.GetDiscoveredData().GetActivationStatus())
	s.Equal(v1.DnfMetadataStatus_UNAVAILABLE, second.GetDiscoveredData().GetDnfMetadataStatus())

	// Verify relay exited cleanly
	err := <-errChan
	s.ErrorIs(err, context.Canceled)
}

// TestRelay_SenderErrorsDoNotStopProcessing verifies that sender errors don't halt the relay
func (s *relayTestSuite) TestRelay_SenderErrorsDoNotStopProcessing() {
	// Sender fails on second report but signals completion
	done := concurrency.NewSignal()
	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex:   1, // fail on second report
		done:          &done,
		expectedCount: 3,
	}

	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			relaytest.NewTestVMReport("100"),
			relaytest.NewTestVMReport("200"),
			relaytest.NewTestVMReport("300"),
		},
	}

	umh := &mockUMH{}
	relay := New(mockIndexReportStream, mockIndexReportSender, umh, 0, 4*time.Hour, 0, 0)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for all reports to be attempted
	select {
	case <-done.Done():
		// All reports attempted
	case <-time.After(1 * time.Second):
		s.Fail("Timeout waiting for reports to be processed")
	}

	cancel()

	// All three messages should have been attempted
	var sentCount int
	concurrency.WithLock(&mockIndexReportSender.mu, func() {
		sentCount = len(mockIndexReportSender.sentMessages)
	})
	s.Require().Equal(3, sentCount)

	err := <-errChan
	s.ErrorIs(err, context.Canceled)

	umh.mu.Lock()
	defer umh.mu.Unlock()
	s.Contains(umh.nacks, "200", "failed send should be recorded as NACK")
}

// TestRelay_ContextCancellation verifies relay stops on context cancellation
func (s *relayTestSuite) TestRelay_ContextCancellation() {
	// The mocked stream signals when first message is sent
	started := concurrency.NewSignal()
	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			relaytest.NewTestVMReport("100"),
			relaytest.NewTestVMReport("200"), // Second message will never be processed
		},
		started: &started,
	}

	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex: -1, // never fail
	}

	relay := New(mockIndexReportStream, mockIndexReportSender, &mockUMH{}, 0, 4*time.Hour, 0, 0)

	ctx, cancel := context.WithCancel(s.ctx)

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for stream to start sending reports
	<-started.Done()

	// Cancel immediately
	cancel()

	// Should exit quickly
	select {
	case err := <-errChan:
		s.ErrorIs(err, context.Canceled)
	case <-time.After(100 * time.Millisecond):
		s.Fail("Relay did not exit after context cancellation")
	}
}

// TestRelay_RateLimiting verifies that rate limiting works
func (s *relayTestSuite) TestRelay_RateLimiting() {
	// Create mock sender
	done := concurrency.NewSignal()
	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex:   -1, // never fail
		done:          &done,
		expectedCount: 1, // only first should pass
	}

	// Send 3 reports from same VSOCK ID
	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			{IndexReport: &v1.IndexReport{VsockCid: "100"}},
			{IndexReport: &v1.IndexReport{VsockCid: "100"}}, // Same ID - should be rate limited
			{IndexReport: &v1.IndexReport{VsockCid: "100"}}, // Same ID - should be rate limited
		},
	}

	// Rate limit: 1 per minute (effectively blocks after first)
	relay := New(mockIndexReportStream, mockIndexReportSender, &mockUMH{}, 1, 4*time.Hour, 0, 0)

	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for first report
	select {
	case <-done.Done():
		// First report processed
	case <-time.After(500 * time.Millisecond):
		s.Fail("Timeout waiting for first report")
	}

	// Give time for other reports to be processed (they should be rate limited)
	time.Sleep(100 * time.Millisecond)

	cancel()

	// Only 1 report should have been sent (others rate limited)
	var sentCount int
	concurrency.WithLock(&mockIndexReportSender.mu, func() {
		sentCount = len(mockIndexReportSender.sentMessages)
	})
	s.Equal(1, sentCount)

	err := <-errChan
	s.ErrorIs(err, context.Canceled)
}

// TestRelay_UMHInteraction verifies the relay observes UMH signals and marks ACKs.
func (s *relayTestSuite) TestRelay_UMHInteraction() {
	done := concurrency.NewSignal()
	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex:   -1,
		done:          &done,
		expectedCount: 1,
	}

	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			{IndexReport: &v1.IndexReport{VsockCid: "100"}},
		},
	}

	umh := &mockUMH{
		retryCh: make(chan string, 1),
	}

	relay := New(mockIndexReportStream, mockIndexReportSender, umh, 0, 4*time.Hour, 0, 0)

	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for the report to be sent.
	select {
	case <-done.Done():
	case <-time.After(time.Second):
		s.Fail("timeout waiting for report send")
	}

	// Verify ObserveSending recorded the VSOCK ID.
	var sends []string
	concurrency.WithLock(&umh.mu, func() {
		sends = append(sends, umh.sends...)
	})
	s.Contains(sends, "100")

	// Send an ACK via UMH and ensure relay records it in cache.
	umh.HandleACK("100")
	time.Sleep(50 * time.Millisecond)

	var cached *cachedReportMetadata
	concurrency.WithLock(&relay.mu, func() {
		cached = relay.cache["100"]
	})
	s.Require().NotNil(cached, "cached report should exist after ACK")
	s.False(cached.lastAckedAt.IsZero(), "lastAckedAt should be recorded")

	cancel()
	<-errChan
}

// TestRelay_StaleCacheEntriesAreEvicted asserts that stale cache entries
// for VSOCK IDs that are no longer active get evicted.
func (s *relayTestSuite) TestRelay_StaleCacheEntriesAreEvicted() {
	const (
		numStale = 25
		numFresh = 25
		total    = numStale + numFresh
	)

	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	threshold := 4 * time.Hour

	evictCh := make(chan time.Time)
	r := &Relay{
		reportStream:      &mockIndexReportStream{},
		reportSender:      &mockIndexReportSender{failOnIndex: -1},
		umh:               &mockUMH{},
		staleAckThreshold: threshold,
		cacheEvictTickCh:  evictCh,
		cache:             make(map[string]*cachedReportMetadata),
		payloadCache:      newReportPayloadCache(0, time.Hour),
	}
	r.umh.OnACK(r.markAcked)

	for i := range numStale {
		id := fmt.Sprintf("stale-%d", i)
		r.cache[id] = &cachedReportMetadata{
			updatedAt: now.Add(-10 * time.Hour),
			limiter:   rate.NewLimiter(1, 1),
		}
	}
	for i := range numFresh {
		id := fmt.Sprintf("fresh-%d", i)
		r.cache[id] = &cachedReportMetadata{
			updatedAt: now.Add(-1 * time.Hour),
			limiter:   rate.NewLimiter(1, 1),
		}
	}
	s.Require().Equal(total, len(r.cache), "precondition: cache should have %d entries", total)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() { errChan <- r.Run(ctx) }()

	evictCh <- now

	cancel()
	<-errChan

	var cacheLen int
	concurrency.WithLock(&r.mu, func() {
		cacheLen = len(r.cache)
	})

	s.Equal(numFresh, cacheLen,
		"cache should contain %d fresh entries but got %d instead", numFresh, cacheLen)
}

// TestRelay_EvictStaleEntries verifies that evictStaleEntries removes cache
// entries whose updatedAt is older than staleAckThreshold, and retains entries
// that are still fresh.
func (s *relayTestSuite) TestRelay_EvictStaleEntries() {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	cases := map[string]struct {
		initialCache map[string]*cachedReportMetadata
		now          time.Time
		threshold    time.Duration
		wantCache    []string
	}{
		"should evict entries older than threshold": {
			now:       now,
			threshold: 4 * time.Hour,
			initialCache: map[string]*cachedReportMetadata{
				"stale": {updatedAt: now.Add(-5 * time.Hour), limiter: rate.NewLimiter(1, 1)},
				"fresh": {updatedAt: now.Add(-1 * time.Hour), limiter: rate.NewLimiter(1, 1)},
			},
			wantCache: []string{"fresh"},
		},
		"should evict stale entry even if it was recently acked": {
			now:       now,
			threshold: 4 * time.Hour,
			initialCache: map[string]*cachedReportMetadata{
				"stale-acked": {
					updatedAt:   now.Add(-10 * time.Hour),
					lastAckedAt: now.Add(-1 * time.Hour),
					limiter:     rate.NewLimiter(1, 1),
				},
				"fresh": {updatedAt: now.Add(-30 * time.Minute), limiter: rate.NewLimiter(1, 1)},
			},
			wantCache: []string{"fresh"},
		},
		"should retain all entries when none are stale": {
			now:       now,
			threshold: 4 * time.Hour,
			initialCache: map[string]*cachedReportMetadata{
				"vm-a": {updatedAt: now.Add(-1 * time.Hour), limiter: rate.NewLimiter(1, 1)},
				"vm-b": {updatedAt: now.Add(-2 * time.Hour), limiter: rate.NewLimiter(1, 1)},
			},
			wantCache: []string{"vm-a", "vm-b"},
		},
		"should evict all entries when all are stale": {
			now:       now,
			threshold: 4 * time.Hour,
			initialCache: map[string]*cachedReportMetadata{
				"old-a": {updatedAt: now.Add(-25 * time.Hour), limiter: rate.NewLimiter(1, 1)},
				"old-b": {updatedAt: now.Add(-48 * time.Hour), limiter: rate.NewLimiter(1, 1)},
			},
			wantCache: nil,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			r := &Relay{
				cache: tc.initialCache,
			}

			r.evictStaleEntries(tc.now, tc.threshold)

			var gotCacheKeys []string
			concurrency.WithLock(&r.mu, func() {
				gotCacheKeys = slices.Collect(maps.Keys(r.cache))
			})

			s.ElementsMatch(gotCacheKeys, tc.wantCache, "cache should contain %v but got %v instead", tc.wantCache, gotCacheKeys)
		})
	}
}

func (s *relayTestSuite) TestRelay_StaleAckAndPayloadSweepTickers() {
	cases := map[string]struct {
		staleAck    time.Duration
		payloadTTL  time.Duration
		wantMeta    bool
		wantPayload bool
	}{
		"non-positive stale ACK and zero payload TTL disable both tickers": {
			staleAck: 0, payloadTTL: 0, wantMeta: false, wantPayload: false,
		},
		"negative stale ACK and zero payload TTL disable both tickers": {
			staleAck: -time.Second, payloadTTL: 0, wantMeta: false, wantPayload: false,
		},
		"non-positive stale ACK but positive payload TTL disables metadata ticker only": {
			staleAck: 0, payloadTTL: time.Hour, wantMeta: false, wantPayload: true,
		},
		"negative stale ACK but positive payload TTL disables metadata ticker only": {
			staleAck: -time.Second, payloadTTL: time.Hour, wantMeta: false, wantPayload: true,
		},
		"positive stale ACK and zero payload TTL enables metadata ticker only": {
			staleAck: time.Hour, payloadTTL: 0, wantMeta: true, wantPayload: false,
		},
		"positive stale ACK and positive payload TTL enables both tickers": {
			staleAck: time.Hour, payloadTTL: time.Hour, wantMeta: true, wantPayload: true,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			r := New(&mockIndexReportStream{}, &mockIndexReportSender{failOnIndex: -1}, &mockUMH{}, 0, tc.staleAck, 0, tc.payloadTTL)
			if tc.wantMeta {
				s.NotNil(r.cacheEvictTicker, "metadata eviction ticker should exist")
				s.NotNil(r.cacheEvictTickCh, "metadata eviction channel should be set")
			} else {
				s.Nil(r.cacheEvictTicker, "metadata eviction ticker should not exist")
				s.Nil(r.cacheEvictTickCh, "metadata eviction channel should be disabled")
			}
			if tc.wantPayload {
				s.NotNil(r.payloadSweepTicker, "payload sweep ticker should exist when payload TTL is positive")
				s.NotNil(r.payloadSweepTickCh, "payload sweep channel should be set when payload TTL is positive")
			} else {
				s.Nil(r.payloadSweepTicker, "payload sweep ticker should not exist when payload TTL is non-positive")
				s.Nil(r.payloadSweepTickCh, "payload sweep channel should be disabled when payload TTL is non-positive")
			}
		})
	}
}

func (s *relayTestSuite) TestRelay_Run_InvokesPayloadSweepWhenStaleAckEvictionDisabled() {
	sweepCh := make(chan time.Time, 1)
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	ttl := 10 * time.Millisecond

	r := &Relay{
		reportStream:       &mockIndexReportStream{},
		reportSender:       &mockIndexReportSender{failOnIndex: -1},
		umh:                &mockUMH{},
		staleAckThreshold:  0,
		payloadCacheTTL:    ttl,
		cacheEvictTickCh:   nil,
		payloadSweepTickCh: sweepCh,
		cache:              make(map[string]*cachedReportMetadata),
		payloadCache:       newReportPayloadCache(4, ttl),
	}
	r.umh.OnACK(r.markAcked)

	vmr := relaytest.NewTestVMReport("100")
	s.Require().Empty(r.payloadCache.Upsert("100", vmr, now.Add(-2*ttl)))
	s.Require().Equal(1, r.payloadCache.Len(), "precondition: one cached payload entry")

	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
	defer cancel()

	sweepCh <- now

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run(ctx)
	}()

	s.Eventually(
		func() bool {
			return r.payloadCache.Len() == 0
		},
		time.Second,
		10*time.Millisecond,
		"expired payload entry should be removed on payload sweep tick",
	)

	cancel()
	s.ErrorIs(<-errCh, context.Canceled)
}

func (s *relayTestSuite) TestRelay_RunReturnsErrorWhenRetryChannelCloses() {
	retryCh := make(chan string)
	close(retryCh)
	umh := &mockUMH{
		retryCh: retryCh,
	}
	r := New(&mockIndexReportStream{}, &mockIndexReportSender{failOnIndex: -1}, umh, 0, 4*time.Hour, 0, 0)
	s.Require().NotNil(r.cacheEvictTicker)

	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()

	err := r.Run(ctx)
	s.Require().Error(err)
	s.ErrorContains(err, "UMH retry command channel closed")
}

func (s *relayTestSuite) TestRelay_RetryCommand_CacheHit_Resends() {
	done := concurrency.NewSignal()
	mockSender := &mockIndexReportSender{
		failOnIndex:   -1,
		done:          &done,
		expectedCount: 1,
	}
	umh := &mockUMH{
		retryCh: make(chan string, 1),
	}
	stream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			relaytest.NewTestVMReport("100"),
		},
	}
	relay := New(stream, mockSender, umh, 0, 4*time.Hour, 4, 24*time.Hour)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- relay.Run(ctx)
	}()

	select {
	case <-done.Done():
	case <-time.After(2 * time.Second):
		s.Fail("timeout waiting for initial send")
	}

	hitBefore := testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("hit"))

	umh.retryCh <- "100"
	time.Sleep(300 * time.Millisecond)

	var sentMessages []*v1.VMReport
	concurrency.WithLock(&mockSender.mu, func() {
		sentMessages = append(sentMessages, mockSender.sentMessages...)
	})
	s.Len(sentMessages, 2)
	s.Equal("100", sentMessages[0].GetIndexReport().GetVsockCid())
	s.Equal("100", sentMessages[1].GetIndexReport().GetVsockCid())

	var sends []string
	concurrency.WithLock(&umh.mu, func() {
		sends = append(sends, umh.sends...)
	})
	s.Len(sends, 2)
	s.Equal([]string{"100", "100"}, sends)

	hitDelta := testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("hit")) - hitBefore
	s.Equal(1.0, hitDelta)

	cancel()
	<-errCh
}

func (s *relayTestSuite) TestRelay_RetryCommand_CacheMiss_IncrementsMissAndDoesNotSend() {
	mockSender := &mockIndexReportSender{
		failOnIndex: -1,
	}
	umh := &mockUMH{
		retryCh: make(chan string, 1),
	}
	stream := &mockIndexReportStream{
		reports: nil,
	}
	relay := New(stream, mockSender, umh, 0, 4*time.Hour, 4, 24*time.Hour)

	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- relay.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	missBefore := testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("miss"))

	umh.retryCh <- "not-cached"
	time.Sleep(200 * time.Millisecond)

	missDelta := testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("miss")) - missBefore
	s.Equal(1.0, missDelta)

	var sentCount int
	concurrency.WithLock(&mockSender.mu, func() {
		sentCount = len(mockSender.sentMessages)
	})
	s.Zero(sentCount)

	cancel()
	<-errCh
}

func (s *relayTestSuite) TestRelay_RetryCommand_CacheDisabled_DoesNotResend() {
	done := concurrency.NewSignal()
	mockSender := &mockIndexReportSender{
		failOnIndex:   -1,
		done:          &done,
		expectedCount: 1,
	}
	umh := &mockUMH{
		retryCh: make(chan string, 1),
	}
	stream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			relaytest.NewTestVMReport("100"),
		},
	}
	relay := New(stream, mockSender, umh, 0, 4*time.Hour, 0, 24*time.Hour)
	s.Equal(0.0, testutil.ToFloat64(metrics.IndexReportCacheSlotsCapacity), "expected cache slots capacity 0 when cache is disabled")
	slotsBase := testutil.ToFloat64(metrics.IndexReportCacheSlotsUsed)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- relay.Run(ctx)
	}()

	select {
	case <-done.Done():
	case <-time.After(2 * time.Second):
		s.Fail("timeout waiting for initial send")
	}

	s.Equal(0, relay.payloadCache.Len(), "expected payload cache to stay empty when disabled")
	s.Equal(slotsBase, testutil.ToFloat64(metrics.IndexReportCacheSlotsUsed), "expected slots used to remain unchanged when cache is disabled")

	missBefore := testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("miss"))
	umh.retryCh <- "100"

	s.Eventually(
		func() bool {
			return testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("miss")) > missBefore
		},
		2*time.Second,
		10*time.Millisecond,
		"expected retry lookup to be processed as cache miss",
	)
	s.Equal(1.0, testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("miss"))-missBefore)

	var sentCount int
	concurrency.WithLock(&mockSender.mu, func() {
		sentCount = len(mockSender.sentMessages)
	})
	s.Equal(1, sentCount)

	var sends []string
	concurrency.WithLock(&umh.mu, func() {
		sends = append(sends, umh.sends...)
	})
	s.Equal([]string{"100"}, sends)

	cancel()
	s.ErrorIs(<-errCh, context.Canceled)
}

func (s *relayTestSuite) TestRelay_ACK_RemovesPayloadCacheEntry() {
	done := concurrency.NewSignal()
	mockSender := &mockIndexReportSender{
		failOnIndex:   -1,
		done:          &done,
		expectedCount: 1,
	}
	umh := &mockUMH{
		retryCh: make(chan string, 1),
	}
	stream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			relaytest.NewTestVMReport("100"),
		},
	}
	relay := New(stream, mockSender, umh, 0, 4*time.Hour, 4, 24*time.Hour)
	slotsBase := testutil.ToFloat64(metrics.IndexReportCacheSlotsUsed)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- relay.Run(ctx)
	}()

	select {
	case <-done.Done():
	case <-time.After(2 * time.Second):
		s.Fail("timeout waiting for send")
	}

	s.Equal(slotsBase+1.0, testutil.ToFloat64(metrics.IndexReportCacheSlotsUsed))

	missBefore := testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("miss"))

	umh.HandleACK("100")
	time.Sleep(100 * time.Millisecond)
	s.Equal(slotsBase+0.0, testutil.ToFloat64(metrics.IndexReportCacheSlotsUsed))

	umh.retryCh <- "100"
	time.Sleep(200 * time.Millisecond)

	missDelta := testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("miss")) - missBefore
	s.Equal(1.0, missDelta)

	var sentCount int
	concurrency.WithLock(&mockSender.mu, func() {
		sentCount = len(mockSender.sentMessages)
	})
	s.Equal(1, sentCount)

	cancel()
	<-errCh
}

func (s *relayTestSuite) TestRelay_RetryCommand_ExpiredPayloadResendsUntilSweepEvicts() {
	ttl := 50 * time.Millisecond
	mockSender := &mockIndexReportSender{
		failOnIndex: -1,
	}
	umh := &mockUMH{}
	relay := &Relay{
		reportSender: mockSender,
		umh:          umh,
		payloadCache: newReportPayloadCache(4, ttl),
	}

	now := time.Now()
	s.Require().Empty(relay.payloadCache.Upsert("100", relaytest.NewTestVMReport("100"), now.Add(-3*ttl)))
	metrics.IndexReportCacheSlotsUsed.Set(float64(relay.payloadCache.Len()))
	s.Equal(1, relay.payloadCache.Len(), "precondition: expired payload should still be present before sweep")

	missBefore := testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("miss"))
	hitBefore := testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("hit"))
	resCountBefore := residencyHistogramSampleCount(s.T())
	lifeCountBefore := lifetimeHistogramSampleCount(s.T())

	relay.handleRetryCommand(s.ctx, "100")

	s.Equal(0.0, testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("miss"))-missBefore)
	s.Equal(1.0, testutil.ToFloat64(metrics.IndexReportCacheLookupsTotal.WithLabelValues("hit"))-hitBefore)
	s.Equal(1, relay.payloadCache.Len(), "expired payload should remain in cache until sweep")
	s.Equal(1.0, testutil.ToFloat64(metrics.IndexReportCacheSlotsUsed))
	var sentCount int
	concurrency.WithLock(&mockSender.mu, func() {
		sentCount = len(mockSender.sentMessages)
	})
	s.Equal(1, sentCount)
	s.Equal(resCountBefore, residencyHistogramSampleCount(s.T()))
	s.Equal(lifeCountBefore, lifetimeHistogramSampleCount(s.T()))

	relay.sweepPayloadCache(now)
	s.Equal(0, relay.payloadCache.Len(), "sweep should evict expired payload")
	s.Equal(0.0, testutil.ToFloat64(metrics.IndexReportCacheSlotsUsed))
	s.Equal(resCountBefore+1, residencyHistogramSampleCount(s.T()))
	s.Equal(lifeCountBefore+1, lifetimeHistogramSampleCount(s.T()))
}

func lifetimeHistogramSampleCount(t *testing.T) uint64 {
	t.Helper()
	return histogramSampleCount(t, "rox_compliance_virtual_machine_relay_index_report_cache_lifetime_seconds")
}

// residencyHistogramSampleCount returns Histogram sample_count for the VM index report payload residency metric.
func residencyHistogramSampleCount(t *testing.T) uint64 {
	t.Helper()
	return histogramSampleCount(t, "rox_compliance_virtual_machine_relay_index_report_cache_residency_seconds")
}

func histogramSampleCount(t *testing.T, wantName string) uint64 {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gathering prometheus metrics should succeed, but failed with: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != wantName {
			continue
		}
		var n uint64
		for _, m := range mf.GetMetric() {
			if h := m.GetHistogram(); h != nil {
				n += h.GetSampleCount()
			}
		}
		return n
	}
	t.Fatalf("metric family %q should exist in default gatherer, but was not found", wantName)
	return 0
}

// Mock implementations

// failingIndexReportStream is a mock IndexReportStream whose Start method
// always fails. It tracks how many times Start is called so tests can assert
// correct behavior.
type failingIndexReportStream struct {
	startCalled int
}

// Start implements IndexReportStream.Start. It always returns a nil channel
// and errTestStreamStart to simulate a stream startup failure.
func (f *failingIndexReportStream) Start(ctx context.Context) (<-chan *v1.VMReport, error) {
	f.startCalled++
	return nil, errTestStreamStart
}

type mockIndexReportStream struct {
	reports []*v1.VMReport
	started *concurrency.Signal // signals when first report is streamed
}

func (m *mockIndexReportStream) Start(ctx context.Context) (<-chan *v1.VMReport, error) {
	reportChan := make(chan *v1.VMReport, len(m.reports))

	go func() {
		for i, report := range m.reports {
			select {
			case <-ctx.Done():
				return
			case reportChan <- report:
				// Signal when first report is streamed
				if i == 0 && m.started != nil {
					m.started.Signal()
				}
			}
		}
	}()

	return reportChan, nil
}

type mockIndexReportSender struct {
	mu            sync.Mutex
	sentMessages  []*v1.VMReport
	failOnIndex   int                 // Index to fail on (0-based), use -1 to never fail
	done          *concurrency.Signal // signals when expectedCount reports are sent
	expectedCount int                 // number of reports expected before signaling done
}

func (m *mockIndexReportSender) Send(_ context.Context, vmReport *v1.VMReport) error {
	var currentIndex int
	concurrency.WithLock(&m.mu, func() {
		currentIndex = len(m.sentMessages)
		m.sentMessages = append(m.sentMessages, vmReport)

		// Signal done when we've sent expected count
		if m.done != nil && len(m.sentMessages) == m.expectedCount {
			m.done.Signal()
		}
	})

	// Fail on the specified index
	if currentIndex == m.failOnIndex {
		return errTest
	}
	return nil
}

// mockUMH is a mock UnconfirmedMessageHandler for testing
type mockUMH struct {
	mu      sync.Mutex
	acks    []string
	nacks   []string
	sends   []string
	retryCh chan string
	onACKCb func(resourceID string)
}

func (m *mockUMH) HandleACK(resourceID string) {
	var cb func(resourceID string)
	concurrency.WithLock(&m.mu, func() {
		m.acks = append(m.acks, resourceID)
		cb = m.onACKCb
	})
	if cb != nil {
		cb(resourceID)
	}
}

func (m *mockUMH) HandleNACK(resourceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nacks = append(m.nacks, resourceID)
}

func (m *mockUMH) ObserveSending(resourceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sends = append(m.sends, resourceID)
}

func (m *mockUMH) RetryCommand() <-chan string {
	if m.retryCh == nil {
		m.retryCh = make(chan string)
	}
	return m.retryCh
}

func (m *mockUMH) OnACK(callback func(resourceID string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onACKCb = callback
}
