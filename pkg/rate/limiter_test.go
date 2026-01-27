package rate

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const workloadName = "test_workload"

// TestClock allows manual time control in tests.
type TestClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewTestClock creates a TestClock starting at the given time.
func NewTestClock(start time.Time) *TestClock {
	return &TestClock{now: start}
}

// Now returns the current frozen time.
func (c *TestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward by the given duration.
func (c *TestClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// mustNewLimiter creates a limiter or fails the test.
func mustNewLimiter(t *testing.T, workloadName string, globalRate float64, bucketCapacity int) *Limiter {
	t.Helper()
	limiter, err := NewLimiter(workloadName, globalRate, bucketCapacity).ForAllWorkloads()
	require.NoError(t, err)
	return limiter
}

// mustNewLimiterWithClock creates a limiter with an injectable clock or fails the test.
func mustNewLimiterWithClock(t *testing.T, workloadName string, globalRate float64, bucketCapacity int, clock Clock) *Limiter {
	t.Helper()
	limiter, err := NewLimiterWithClock(workloadName, globalRate, bucketCapacity, clock).ForAllWorkloads()
	require.NoError(t, err)
	return limiter
}

func TestNewLimiter(t *testing.T) {
	t.Run("should create a new limiter", func(t *testing.T) {
		limiter, err := NewLimiter(workloadName, 10.0, 50).ForAllWorkloads()
		require.NoError(t, err)
		assert.Equal(t, 10.0, limiter.GlobalRate())
		assert.Equal(t, 50, limiter.BucketCapacity())
		assert.Equal(t, workloadName, limiter.WorkloadName())
	})
	t.Run("should create a new limiter with rate limiting disabled", func(t *testing.T) {
		limiter, err := NewLimiter(workloadName, 0.0, 1).ForAllWorkloads() // rate=0 disables limiting, bucketCapacity still must be >= 1
		require.NoError(t, err)
		assert.Equal(t, 0.0, limiter.GlobalRate())
		assert.Equal(t, 1, limiter.BucketCapacity())
		assert.Equal(t, workloadName, limiter.WorkloadName())
	})
	t.Run("should create a new limiter with rate higher than bucket capacity", func(t *testing.T) {
		limiter, err := NewLimiter(workloadName, 50.0, 2).ForAllWorkloads()
		require.NoError(t, err)
		assert.Equal(t, 50.0, limiter.GlobalRate())
		assert.Equal(t, 2, limiter.BucketCapacity())
		assert.Equal(t, workloadName, limiter.WorkloadName())
	})
	t.Run("should error on empty workload name", func(t *testing.T) {
		_, err := NewLimiter("", 10.0, 50).ForAllWorkloads()
		assert.ErrorIs(t, err, ErrEmptyWorkloadName)
	})
	t.Run("should error on negative rate", func(t *testing.T) {
		_, err := NewLimiter(workloadName, -1.0, 50).ForAllWorkloads()
		assert.ErrorIs(t, err, ErrNegativeRate)
	})
	t.Run("should error on zero bucket capacity", func(t *testing.T) {
		_, err := NewLimiter(workloadName, 10.0, 0).ForAllWorkloads()
		assert.ErrorIs(t, err, ErrInvalidBucketCapacity)
	})
}

func TestLimiterOption_ForWorkload(t *testing.T) {
	tests := map[string]struct {
		workloadName    string
		globalRate      float64
		bucketCapacity  int
		acceptsFn       func(msg *central.MsgFromSensor) bool
		expectedErr     error
		expectedErrText string
		expectLimiter   bool
		shouldSkipCheck bool
	}{
		"should return error when acceptsFn is nil": {
			workloadName:    workloadName,
			globalRate:      1,
			bucketCapacity:  1,
			acceptsFn:       nil,
			expectedErrText: "acceptsFn must not be nil",
		},
		"should return error when limiter creation fails": {
			workloadName:   "",
			globalRate:     1,
			bucketCapacity: 1,
			acceptsFn: func(msg *central.MsgFromSensor) bool {
				return msg != nil
			},
			expectedErr: ErrEmptyWorkloadName,
		},
		"should allow configuring a workload filter": {
			workloadName:   workloadName,
			globalRate:     1,
			bucketCapacity: 1,
			acceptsFn: func(msg *central.MsgFromSensor) bool {
				return msg != nil
			},
			expectLimiter:   true,
			shouldSkipCheck: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			limiter, err := NewLimiter(tt.workloadName, tt.globalRate, tt.bucketCapacity).ForWorkload(tt.acceptsFn)
			if tt.expectedErr != nil || tt.expectedErrText != "" {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
				if tt.expectedErrText != "" {
					assert.EqualError(t, err, tt.expectedErrText)
				}
				assert.Nil(t, limiter)
				return
			}
			require.NoError(t, err)
			if tt.expectLimiter {
				assert.NotNil(t, limiter)
			}
			if !tt.shouldSkipCheck {
				return
			}

			allowed, reason := limiter.TryConsume("client-1", &central.MsgFromSensor{})
			require.True(t, allowed)
			assert.Empty(t, reason)

			allowed, reason = limiter.TryConsume("client-1", &central.MsgFromSensor{})
			assert.False(t, allowed)
			assert.Equal(t, ReasonRateLimitExceeded, reason)

			allowed, reason = limiter.TryConsume("client-1", nil)
			assert.True(t, allowed)
			assert.Empty(t, reason)
		})
	}
}

func TestTryConsume_Disabled(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 0, 5)
	for i := range 100 {
		allowed, reason := limiter.TryConsume("test-cluster", nil)
		assert.True(t, allowed, "request %d should be allowed when rate limiting is disabled", i)
		assert.Empty(t, reason)
	}
}

func TestTryConsume_SingleClient(t *testing.T) {
	clock := NewTestClock(time.Now())
	limiter := mustNewLimiterWithClock(t, "test", 10, 50, clock) // rate=10 req/s, bucket capacity=50

	// With 1 client, per-client burst = 50/1 = 50 requests
	for i := range 50 {
		allowed, reason := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "request %d should be allowed within burst", i)
		assert.Empty(t, reason)
	}

	// Time is frozen - no new tokens can be added between requests.
	// 51st request should be rejected (burst exhausted)
	allowed, reason := limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "request should be rejected after burst exhausted")
	assert.Equal(t, "rate limit exceeded", reason)

	// Advance time and verify tokens refill correctly
	clock.Advance(500 * time.Millisecond) // At 10 req/s, expect ~5 tokens
	for i := range 5 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "request %d should be allowed after time advance", i)
	}
	allowed, _ = limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "should be rejected after consuming refilled tokens")
}

func TestTryConsume_MultipleClients_Fairness(t *testing.T) {
	clock := NewTestClock(time.Now())
	limiter := mustNewLimiterWithClock(t, "test", 12, 60, clock) // rate=12 req/s, bucket capacity=60

	client1 := "client-1"
	client2 := "client-2"
	client3 := "client-3"

	// Create all clients first to establish fair rates upfront
	// With 3 clients, each gets per-client burst = 60/3 = 20
	limiter.getOrCreateLimiter(client1)
	limiter.getOrCreateLimiter(client2)
	limiter.getOrCreateLimiter(client3)

	// Exhaust burst for client-1 (20 requests)
	for i := range 20 {
		allowed, _ := limiter.TryConsume(client1, nil)
		assert.True(t, allowed, "client-1 request %d should be allowed", i)
	}
	allowed, _ := limiter.TryConsume(client1, nil)
	assert.False(t, allowed, "client-1 should be rate limited after burst")

	// client-2 and client-3 should still have their full burst capacity
	for i := range 20 {
		allowed, _ := limiter.TryConsume(client2, nil)
		assert.True(t, allowed, "client-2 request %d should be allowed", i)
	}
	for i := range 20 {
		allowed, _ := limiter.TryConsume(client3, nil)
		assert.True(t, allowed, "client-3 request %d should be allowed", i)
	}

	// All clients exhausted - time is frozen so no tokens refilled
	allowed, _ = limiter.TryConsume(client2, nil)
	assert.False(t, allowed, "client-2 should be rate limited")
	allowed, _ = limiter.TryConsume(client3, nil)
	assert.False(t, allowed, "client-3 should be rate limited")
}

func TestTryConsume_Rebalancing(t *testing.T) {
	clock := NewTestClock(time.Now())
	// rate=10 req/s, bucket capacity=100
	limiter := mustNewLimiterWithClock(t, workloadName, 10, 100, clock)

	// Start with client-1: per-client burst = 100/1 = 100
	for i := range 100 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "client-1 initial request %d should be allowed", i)
	}
	allowed, _ := limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "client-1 should be rate limited after initial burst")

	// Add client-2: rebalances to per-client burst = 100/2 = 50
	// client-2 gets fresh bucket with capacity 50
	for i := range 50 {
		allowed, _ := limiter.TryConsume("client-2", nil)
		assert.True(t, allowed, "client-2 request %d should be allowed after rebalancing", i)
	}
	allowed, _ = limiter.TryConsume("client-2", nil)
	assert.False(t, allowed, "client-2 should be rate limited after burst")

	// Advance time for token refill (at 5 req/s per client, 7 tokens refill in 1.4 seconds)
	clock.Advance(1500 * time.Millisecond)

	// Both clients should get ~7 tokens back (5 req/s * 1.5s)
	// Verify we can make a few requests
	for range 5 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "client-1 should have refilled tokens")
		allowed, _ = limiter.TryConsume("client-2", nil)
		assert.True(t, allowed, "client-2 should have refilled tokens")
	}
}

func TestTryConsume_BurstWindow(t *testing.T) {
	clock := NewTestClock(time.Now())
	// rate=10 req/s, bucket capacity=100
	limiter := mustNewLimiterWithClock(t, "test", 10, 100, clock)

	// With 1 client, per-client burst = 100/1 = 100
	for i := range 100 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "request %d should be allowed within burst", i)
	}

	// 101st request rejected
	allowed, _ := limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "request should be rejected after burst exhausted")

	// Advance time for refill (at 10 req/s, 15 tokens refill in 1.5 seconds)
	clock.Advance(1500 * time.Millisecond)

	// Should get ~15 tokens back - verify we can make at least 10 requests
	for range 10 {
		allowed, _ = limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "should have refilled tokens")
	}
}

func TestPerClientBurst(t *testing.T) {
	tests := map[string]struct {
		bucketCapacity                  int
		numClients                      int
		expectedPerClientBucketCapacity int
	}{
		"should calculate burst correctly for single client": {
			bucketCapacity:                  50,
			numClients:                      1,
			expectedPerClientBucketCapacity: 50, // 50/1 = 50
		},
		"should calculate burst correctly for multiple clients": {
			bucketCapacity:                  60,
			numClients:                      3,
			expectedPerClientBucketCapacity: 20, // 60/3 = 20
		},
		"should return minimum bucket capacity of 1 for many clients": {
			bucketCapacity:                  5,
			numClients:                      10,
			expectedPerClientBucketCapacity: 1, // 5/10 = 0, but min is 1
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			limiter := mustNewLimiter(t, "test", 10, tt.bucketCapacity)
			burst := limiter.perClientBucketCapacity(tt.numClients)
			assert.Equal(t, tt.expectedPerClientBucketCapacity, burst)
		})
	}
}

func TestRebalancing_DynamicClientCount(t *testing.T) {
	clock := NewTestClock(time.Now())
	// rate=30 req/s, bucket capacity=300
	limiter := mustNewLimiterWithClock(t, "test", 30, 300, clock)

	// Client 1: gets 30/1 = 30 req/s, burst = 30*10s = 300
	for range 300 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed)
	}

	// Add client 2: rebalances to 30/2 = 15 req/s each, burst = 15*10s = 150
	for range 150 {
		allowed, _ := limiter.TryConsume("client-2", nil)
		assert.True(t, allowed)
	}

	// Add client 3: rebalances to 30/3 = 10 req/s each, burst = 10*10s = 100
	for range 100 {
		allowed, _ := limiter.TryConsume("client-3", nil)
		assert.True(t, allowed)
	}

	// All clients should be limited now - time is frozen
	allowed, _ := limiter.TryConsume("client-1", nil)
	assert.False(t, allowed)
	allowed, _ = limiter.TryConsume("client-2", nil)
	assert.False(t, allowed)
	allowed, _ = limiter.TryConsume("client-3", nil)
	assert.False(t, allowed)

	// Advance time for token refill (at 10 req/s per client, 15 tokens refill in 1.5s)
	clock.Advance(1500 * time.Millisecond)

	// Each client should get ~15 tokens back - verify at least 10 work
	for range 10 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "client-1 should get tokens after refill")
		allowed, _ = limiter.TryConsume("client-2", nil)
		assert.True(t, allowed, "client-2 should get tokens after refill")
		allowed, _ = limiter.TryConsume("client-3", nil)
		assert.True(t, allowed, "client-3 should get tokens after refill")
	}
}

func TestOnClientDisconnect(t *testing.T) {
	clock := NewTestClock(time.Now())
	limiter := mustNewLimiterWithClock(t, "test", 20, 100, clock) // rate=20 req/s, bucket capacity=100

	// Create 2 clients: each gets 20/2 = 10 req/s, burst = 10*5s = 50
	limiter.getOrCreateLimiter("client-1")
	limiter.getOrCreateLimiter("client-2")

	// Exhaust client-1's burst
	for i := 0; i < 50; i++ {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed)
	}
	allowed, _ := limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "client-1 should be limited after burst")

	// Disconnect client-2
	limiter.OnClientDisconnect("client-2")

	// client-1 should now get full rate: 20/1 = 20 req/s, burst = 20*5s = 100
	// The existing limiter's burst is updated to 100, but tokens were exhausted
	// Just verify the client-2 is removed
	assert.Equal(t, 1, limiter.numActiveClients())

	// Verify client-2 is no longer tracked
	exists := concurrency.WithRLock1(&limiter.mu, func() bool {
		_, ok := limiter.buckets["client-2"]
		return ok
	})
	assert.False(t, exists, "client-2 should be removed from buckets")
}

func TestOnClientDisconnect_DisabledRateLimiter(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 0, 50)

	// Should be a no-op when rate limiting is disabled
	limiter.OnClientDisconnect("client-1")
	assert.Equal(t, 0, limiter.numActiveClients())
}

func TestOnClientDisconnect_NonexistentClient(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 10, 50)

	// Create one client
	limiter.getOrCreateLimiter("client-1")

	// Disconnect a client that was never connected - should be a no-op
	limiter.OnClientDisconnect("nonexistent-client")
	assert.Equal(t, 1, limiter.numActiveClients())
}
