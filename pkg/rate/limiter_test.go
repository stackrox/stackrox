package rate

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const workloadName = "test_workload"

// mustNewLimiter creates a limiter or fails the test.
func mustNewLimiter(t *testing.T, workloadName string, globalRate float64, bucketCapacity int) *Limiter {
	t.Helper()
	limiter, err := NewLimiter(workloadName, globalRate, bucketCapacity).ForAllWorkloads()
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
		limiter, err := NewLimiter(workloadName, 0.0, 1).ForAllWorkloads()
		require.NoError(t, err)
		assert.Equal(t, 0.0, limiter.GlobalRate())
		assert.Equal(t, 1, limiter.BucketCapacity())
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

			// With capacity=1, first consume should succeed.
			allowed, reason := limiter.TryConsume("client-1", &central.MsgFromSensor{})
			require.True(t, allowed)
			assert.Empty(t, reason)

			// Second consume should be rejected (no tokens left).
			allowed, reason = limiter.TryConsume("client-1", &central.MsgFromSensor{})
			assert.False(t, allowed)
			assert.Equal(t, ReasonRateLimitExceeded, reason)

			// nil message should pass through (not accepted by filter).
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
	limiter := mustNewLimiter(t, "test", 10, 50) // capacity=50

	// With 1 client, per-client capacity = 50/1 = 50 tokens
	for i := range 50 {
		allowed, reason := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "request %d should be allowed within capacity", i)
		assert.Empty(t, reason)
	}

	// 51st request should be rejected (all tokens consumed, none returned)
	allowed, reason := limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "request should be rejected after capacity exhausted")
	assert.Equal(t, ReasonRateLimitExceeded, reason)
}

func TestReturn_RefillsTokens(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 1, 5) // capacity=5

	// Exhaust all tokens
	for range 5 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		require.True(t, allowed)
	}
	allowed, _ := limiter.TryConsume("client-1", nil)
	require.False(t, allowed, "should be rejected after capacity exhausted")

	// Return 3 tokens
	for range 3 {
		limiter.Return("client-1", nil)
	}

	// Should be able to consume 3 more
	for i := range 3 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "request %d should be allowed after return", i)
	}

	// 4th should be rejected
	allowed, _ = limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "should be rejected after re-exhausted")
}

func TestReturn_CapsAtCapacity(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 1, 3) // capacity=3

	// Consume 1 token, then return 10 times — should never exceed capacity
	allowed, _ := limiter.TryConsume("client-1", nil)
	require.True(t, allowed)

	for range 10 {
		limiter.Return("client-1", nil)
	}

	// Should be able to consume exactly 3 (capacity), not more
	for i := range 3 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "request %d should be allowed", i)
	}
	allowed, _ = limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "should be rejected at capacity")
}

func TestReturn_DisconnectedClient(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 1, 5)

	// Consume a token
	allowed, _ := limiter.TryConsume("client-1", nil)
	require.True(t, allowed)

	// Disconnect client
	limiter.OnClientDisconnect("client-1")

	// Return should be a no-op (no panic)
	assert.NotPanics(t, func() {
		limiter.Return("client-1", nil)
	})
}

func TestReturn_WorkloadFilter(t *testing.T) {
	limiter, err := NewLimiter("test", 1, 2).ForWorkload(func(msg *central.MsgFromSensor) bool {
		return msg != nil
	})
	require.NoError(t, err)

	// Consume both tokens with non-nil messages
	allowed, _ := limiter.TryConsume("client-1", &central.MsgFromSensor{})
	require.True(t, allowed)
	allowed, _ = limiter.TryConsume("client-1", &central.MsgFromSensor{})
	require.True(t, allowed)

	// Return with nil message should be a no-op (doesn't match filter)
	limiter.Return("client-1", nil)

	// Still should be rejected (token not actually returned)
	allowed, _ = limiter.TryConsume("client-1", &central.MsgFromSensor{})
	assert.False(t, allowed, "nil return should not refill tokens")

	// Return with non-nil message should actually refill
	limiter.Return("client-1", &central.MsgFromSensor{})

	allowed, _ = limiter.TryConsume("client-1", &central.MsgFromSensor{})
	assert.True(t, allowed, "non-nil return should refill tokens")
}

func TestTryConsume_MultipleClients_Fairness(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 12, 60) // capacity=60

	client1 := "client-1"
	client2 := "client-2"
	client3 := "client-3"

	// Force creation of all 3 clients via TryConsume (first consume for each).
	// After 3 clients, each gets per-client capacity = 60/3 = 20.
	// First consume for each client uses one token.
	allowed, _ := limiter.TryConsume(client1, nil)
	require.True(t, allowed)
	allowed, _ = limiter.TryConsume(client2, nil)
	require.True(t, allowed)
	allowed, _ = limiter.TryConsume(client3, nil)
	require.True(t, allowed)

	// Return those initial tokens to reset to full capacity.
	limiter.Return(client1, nil)
	limiter.Return(client2, nil)
	limiter.Return(client3, nil)

	// Exhaust capacity for client-1 (20 requests)
	for i := range 20 {
		allowed, _ := limiter.TryConsume(client1, nil)
		assert.True(t, allowed, "client-1 request %d should be allowed", i)
	}
	allowed, _ = limiter.TryConsume(client1, nil)
	assert.False(t, allowed, "client-1 should be limited after capacity exhausted")

	// client-2 and client-3 should still have their full capacity
	for i := range 20 {
		allowed, _ := limiter.TryConsume(client2, nil)
		assert.True(t, allowed, "client-2 request %d should be allowed", i)
	}
	for i := range 20 {
		allowed, _ := limiter.TryConsume(client3, nil)
		assert.True(t, allowed, "client-3 request %d should be allowed", i)
	}

	// All clients exhausted
	allowed, _ = limiter.TryConsume(client2, nil)
	assert.False(t, allowed, "client-2 should be limited")
	allowed, _ = limiter.TryConsume(client3, nil)
	assert.False(t, allowed, "client-3 should be limited")
}

func TestTryConsume_Rebalancing(t *testing.T) {
	limiter := mustNewLimiter(t, workloadName, 10, 100) // capacity=100

	// Start with client-1: per-client capacity = 100/1 = 100
	for i := range 100 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "client-1 initial request %d should be allowed", i)
	}
	allowed, _ := limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "client-1 should be limited after capacity exhausted")

	// Add client-2: rebalances to per-client capacity = 100/2 = 50
	// client-2 gets fresh bucket with capacity 50
	for i := range 50 {
		allowed, _ := limiter.TryConsume("client-2", nil)
		assert.True(t, allowed, "client-2 request %d should be allowed after rebalancing", i)
	}
	allowed, _ = limiter.TryConsume("client-2", nil)
	assert.False(t, allowed, "client-2 should be limited after capacity exhausted")

	// No time-based refill: both clients stay limited until Return is called.
	allowed, _ = limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "client-1 should still be limited (no time-based refill)")
	allowed, _ = limiter.TryConsume("client-2", nil)
	assert.False(t, allowed, "client-2 should still be limited (no time-based refill)")

	// Return tokens and verify they're available again
	for range 5 {
		limiter.Return("client-1", nil)
		limiter.Return("client-2", nil)
	}
	for range 5 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "client-1 should have returned tokens")
		allowed, _ = limiter.TryConsume("client-2", nil)
		assert.True(t, allowed, "client-2 should have returned tokens")
	}
}

func TestReturn_ConsumeReturnCycle(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 1, 2) // capacity=2

	// Simulate processing: consume, process, return, repeat
	for cycle := range 100 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "cycle %d: should be allowed", cycle)
		limiter.Return("client-1", nil)
	}

	// After 100 cycles, bucket should be at full capacity
	for i := range 2 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "final burst request %d should be allowed", i)
	}
	allowed, _ := limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "should be limited after capacity")
}

func TestReturn_ConcurrentConsumeReturn(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 1, 10) // capacity=10

	var wg sync.WaitGroup
	iterations := 1000

	// Spawn multiple goroutines that consume and return tokens concurrently.
	for g := range 5 {
		wg.Add(1)
		go func(clientID string) {
			defer wg.Done()
			for range iterations {
				if allowed, _ := limiter.TryConsume(clientID, nil); allowed {
					limiter.Return(clientID, nil)
				}
			}
		}(fmt.Sprintf("client-%d", g))
	}

	wg.Wait()

	// After all goroutines complete, all tokens should be returned.
	// Each client should have its full capacity available.
	assert.Equal(t, 5, limiter.numActiveClients())
	for g := range 5 {
		clientID := fmt.Sprintf("client-%d", g)
		capacity := limiter.perClientBucketCapacity(5) // 10/5 = 2
		for i := range capacity {
			allowed, _ := limiter.TryConsume(clientID, nil)
			assert.True(t, allowed, "%s request %d should be allowed after concurrent test", clientID, i)
		}
		allowed, _ := limiter.TryConsume(clientID, nil)
		assert.False(t, allowed, "%s should be limited at capacity", clientID)
	}
}

func TestPerClientBurst(t *testing.T) {
	tests := map[string]struct {
		bucketCapacity                  int
		numClients                      int
		expectedPerClientBucketCapacity int
	}{
		"should calculate capacity correctly for single client": {
			bucketCapacity:                  50,
			numClients:                      1,
			expectedPerClientBucketCapacity: 50,
		},
		"should calculate capacity correctly for multiple clients": {
			bucketCapacity:                  60,
			numClients:                      3,
			expectedPerClientBucketCapacity: 20,
		},
		"should return minimum capacity of 1 for many clients": {
			bucketCapacity:                  5,
			numClients:                      10,
			expectedPerClientBucketCapacity: 1,
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
	limiter := mustNewLimiter(t, "test", 30, 300) // capacity=300

	// Client 1: gets capacity = 300/1 = 300
	for range 300 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed)
	}

	// Add client 2: rebalances to capacity = 300/2 = 150
	for range 150 {
		allowed, _ := limiter.TryConsume("client-2", nil)
		assert.True(t, allowed)
	}

	// Add client 3: rebalances to capacity = 300/3 = 100
	for range 100 {
		allowed, _ := limiter.TryConsume("client-3", nil)
		assert.True(t, allowed)
	}

	// All clients should be limited now (no time-based refill)
	allowed, _ := limiter.TryConsume("client-1", nil)
	assert.False(t, allowed)
	allowed, _ = limiter.TryConsume("client-2", nil)
	assert.False(t, allowed)
	allowed, _ = limiter.TryConsume("client-3", nil)
	assert.False(t, allowed)

	// Return tokens and verify they're usable
	for range 10 {
		limiter.Return("client-1", nil)
		limiter.Return("client-2", nil)
		limiter.Return("client-3", nil)
	}
	for range 10 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed, "client-1 should get tokens after return")
		allowed, _ = limiter.TryConsume("client-2", nil)
		assert.True(t, allowed, "client-2 should get tokens after return")
		allowed, _ = limiter.TryConsume("client-3", nil)
		assert.True(t, allowed, "client-3 should get tokens after return")
	}
}

func TestOnClientDisconnect_Nil(t *testing.T) {
	limiter, err := NewLimiter(workloadName, -1.0, -2.0).ForAllWorkloads()
	assert.Error(t, err)
	assert.Nil(t, limiter)
	assert.NotPanics(t, func() {
		limiter.OnClientDisconnect("client-2")
		limiter.TryConsume("client-2", nil)
		limiter.Return("client-2", nil)
	})
}

func TestOnClientDisconnect(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 20, 100) // capacity=100

	// Force creation of 2 clients: each gets 100/2 = 50
	allowed, _ := limiter.TryConsume("client-1", nil)
	require.True(t, allowed)
	allowed, _ = limiter.TryConsume("client-2", nil)
	require.True(t, allowed)

	// Return initial tokens
	limiter.Return("client-1", nil)
	limiter.Return("client-2", nil)

	// Exhaust client-1's capacity (50)
	for i := 0; i < 50; i++ {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.True(t, allowed)
	}
	allowed, _ = limiter.TryConsume("client-1", nil)
	assert.False(t, allowed, "client-1 should be limited after capacity exhausted")

	// Disconnect client-2
	limiter.OnClientDisconnect("client-2")
	assert.Equal(t, 1, limiter.numActiveClients())

	// client-2 should be removed
	limiter.mu.Lock()
	_, exists := limiter.buckets["client-2"]
	limiter.mu.Unlock()
	assert.False(t, exists, "client-2 should be removed from buckets")

	// client-1's capacity is now 100/1 = 100.
	// client-1 already has 50 tokens in-flight, so available was capped at min(0, 100) = 0.
	// But capacity increased, so returning the 50 in-flight tokens will make 50 available,
	// which is within the new capacity of 100.
}

func TestOnClientDisconnect_DisabledRateLimiter(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 0, 50)

	limiter.OnClientDisconnect("client-1")
	assert.Equal(t, 0, limiter.numActiveClients())
}

func TestOnClientDisconnect_NonexistentClient(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 10, 50)

	// Create one client
	limiter.TryConsume("client-1", nil)

	// Disconnect a client that was never connected - should be a no-op
	limiter.OnClientDisconnect("nonexistent-client")
	assert.Equal(t, 1, limiter.numActiveClients())
}

func TestNoTimeBasedRefill(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 10, 5) // capacity=5

	// Consume all tokens
	for range 5 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		require.True(t, allowed)
	}

	// Without Return, tokens never come back (no time-based refill)
	for range 100 {
		allowed, _ := limiter.TryConsume("client-1", nil)
		assert.False(t, allowed, "should stay limited without Return calls")
	}

	// Only Return restores capacity
	limiter.Return("client-1", nil)
	allowed, _ := limiter.TryConsume("client-1", nil)
	assert.True(t, allowed, "should be allowed after explicit Return")
}

func TestReturn_NilLimiter(t *testing.T) {
	var limiter *Limiter
	assert.NotPanics(t, func() {
		limiter.Return("client-1", nil)
	})
}
