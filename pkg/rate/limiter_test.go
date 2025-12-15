package rate

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const workloadName = "test_workload"

// mustNewLimiter creates a limiter or fails the test.
func mustNewLimiter(t *testing.T, workloadName string, globalRate float64, bucketCapacity int) *Limiter {
	t.Helper()
	limiter, err := NewLimiter(workloadName, globalRate, bucketCapacity)
	require.NoError(t, err)
	return limiter
}

func TestNewLimiter(t *testing.T) {
	t.Run("should create a new limiter", func(t *testing.T) {
		limiter, err := NewLimiter(workloadName, 10.0, 50)
		require.NoError(t, err)
		assert.Equal(t, 10.0, limiter.GlobalRate())
		assert.Equal(t, 50, limiter.BucketCapacity())
		assert.Equal(t, workloadName, limiter.WorkloadName())
	})
	t.Run("should create a new limiter with rate limiting disabled", func(t *testing.T) {
		limiter, err := NewLimiter(workloadName, 0.0, 1) // rate=0 disables limiting, bucketCapacity still must be >= 1
		require.NoError(t, err)
		assert.Equal(t, 0.0, limiter.GlobalRate())
		assert.Equal(t, 1, limiter.BucketCapacity())
		assert.Equal(t, workloadName, limiter.WorkloadName())
	})
	t.Run("should create a new limiter with rate higher than bucket capacity", func(t *testing.T) {
		limiter, err := NewLimiter(workloadName, 50.0, 2)
		require.NoError(t, err)
		assert.Equal(t, 50.0, limiter.GlobalRate())
		assert.Equal(t, 2, limiter.BucketCapacity())
		assert.Equal(t, workloadName, limiter.WorkloadName())
	})
	t.Run("should error on empty workload name", func(t *testing.T) {
		_, err := NewLimiter("", 10.0, 50)
		assert.ErrorIs(t, err, ErrEmptyWorkloadName)
	})
	t.Run("should error on negative rate", func(t *testing.T) {
		_, err := NewLimiter(workloadName, -1.0, 50)
		assert.ErrorIs(t, err, ErrNegativeRate)
	})
	t.Run("should error on zero bucket capacity", func(t *testing.T) {
		_, err := NewLimiter(workloadName, 10.0, 0)
		assert.ErrorIs(t, err, ErrInvalidBucketCapacity)
	})
}

func TestTryConsume_Disabled(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 0, 5)
	for i := range 100 {
		allowed, reason := limiter.TryConsume("test-cluster")
		assert.True(t, allowed, "request %d should be allowed when rate limiting is disabled", i)
		assert.Empty(t, reason)
	}
}

func TestTryConsume_SingleSensor(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 10, 50) // rate=10 req/s, bucket capacity=50

	// With 1 sensor, per-sensor burst = 50/1 = 50 requests
	for i := range 50 {
		allowed, reason := limiter.TryConsume("sensor-1")
		assert.True(t, allowed, "request %d should be allowed within burst", i)
		assert.Empty(t, reason)
	}

	// TODO: FLAKE MAGNET: what if new token is added to the bucket?

	// 51st request should be rejected (burst exhausted)
	allowed, reason := limiter.TryConsume("sensor-1")
	assert.False(t, allowed, "request should be rejected after burst exhausted")
	assert.Equal(t, "rate limit exceeded", reason)
}

func TestTryConsume_MultipleSensors_Fairness(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 12, 60) // rate=12 req/s, bucket capacity=60

	sensor1 := "sensor-1"
	sensor2 := "sensor-2"
	sensor3 := "sensor-3"

	// Create all sensors first to establish fair rates upfront
	// With 3 sensors, each gets per-sensor burst = 60/3 = 20
	limiter.getOrCreateLimiter(sensor1)
	limiter.getOrCreateLimiter(sensor2)
	limiter.getOrCreateLimiter(sensor3)

	// Exhaust burst for sensor-1 (20 requests)
	for i := range 20 {
		allowed, _ := limiter.TryConsume(sensor1)
		assert.True(t, allowed, "sensor-1 request %d should be allowed", i)
	}
	allowed, _ := limiter.TryConsume(sensor1)
	assert.False(t, allowed, "sensor-1 should be rate limited after burst")

	// Sensor-2 and sensor-3 should still have their full burst capacity
	for i := range 20 {
		allowed, _ := limiter.TryConsume(sensor2)
		assert.True(t, allowed, "sensor-2 request %d should be allowed", i)
	}
	for i := range 20 {
		allowed, _ := limiter.TryConsume(sensor3)
		assert.True(t, allowed, "sensor-3 request %d should be allowed", i)
	}

	// All sensors exhausted
	allowed, _ = limiter.TryConsume(sensor2)
	assert.False(t, allowed, "sensor-2 should be rate limited")
	allowed, _ = limiter.TryConsume(sensor3)
	assert.False(t, allowed, "sensor-3 should be rate limited")
}

func TestTryConsume_Rebalancing(t *testing.T) {
	// Set test deadline to fail fast if something goes wrong
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// rate=10 req/s, bucket capacity=100
	limiter := mustNewLimiter(t, workloadName, 10, 100)

	// Start with sensor-1: per-sensor burst = 100/1 = 100
	for i := range 100 {
		allowed, _ := limiter.TryConsume("sensor-1")
		assert.True(t, allowed, "sensor-1 initial request %d should be allowed", i)
	}
	allowed, _ := limiter.TryConsume("sensor-1")
	assert.False(t, allowed, "sensor-1 should be rate limited after initial burst")

	// Add sensor-2: rebalances to per-sensor burst = 100/2 = 50
	// sensor-2 gets fresh bucket with capacity 50
	for i := range 50 {
		allowed, _ := limiter.TryConsume("sensor-2")
		assert.True(t, allowed, "sensor-2 request %d should be allowed after rebalancing", i)
	}
	allowed, _ = limiter.TryConsume("sensor-2")
	assert.False(t, allowed, "sensor-2 should be rate limited after burst")

	// Wait for token refill (at 5 req/s per sensor, ~7 tokens refill in 1.5 seconds)
	waitDuration := 1500 * time.Millisecond
	select {
	case <-time.After(waitDuration):
		// Normal path - waited successfully
	case <-ctx.Done():
		require.Fail(t, "test deadline exceeded while waiting for token refill")
		return
	}

	// Both sensors should get ~7-8 tokens back (5 req/s * 1.5s)
	// Just verify we can make a few requests
	for range 5 {
		allowed, _ := limiter.TryConsume("sensor-1")
		assert.True(t, allowed, "sensor-1 should have refilled tokens")
		allowed, _ = limiter.TryConsume("sensor-2")
		assert.True(t, allowed, "sensor-2 should have refilled tokens")
	}
}

func TestTryConsume_BurstWindow(t *testing.T) {
	// Set test deadline to fail fast if something goes wrong
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// rate=10 req/s, bucket capacity=100
	limiter := mustNewLimiter(t, "test", 10, 100)

	// With 1 sensor, per-sensor burst = 100/1 = 100
	for i := range 100 {
		allowed, _ := limiter.TryConsume("sensor-1")
		assert.True(t, allowed, "request %d should be allowed within burst", i)
	}

	// 101st request rejected
	allowed, _ := limiter.TryConsume("sensor-1")
	assert.False(t, allowed, "request should be rejected after burst exhausted")

	// Wait for refill (at 10 req/s, ~15 tokens refill in 1.5 second)
	waitDuration := 1500 * time.Millisecond
	select {
	case <-time.After(waitDuration):
		// Normal path - waited successfully
	case <-ctx.Done():
		require.Fail(t, "test deadline exceeded during sleep")
		return
	}

	// Should get ~15 tokens back - verify we can make at least 10 requests
	for range 10 {
		allowed, _ = limiter.TryConsume("sensor-1")
		assert.True(t, allowed, "should have refilled tokens")
	}
}

func TestPerSensorBurst(t *testing.T) {
	tests := map[string]struct {
		bucketCapacity                  int
		numSensors                      int
		expectedPerSensorBucketCapacity int
	}{
		"should calculate burst correctly for single sensor": {
			bucketCapacity:                  50,
			numSensors:                      1,
			expectedPerSensorBucketCapacity: 50, // 50/1 = 50
		},
		"should calculate burst correctly for multiple sensors": {
			bucketCapacity:                  60,
			numSensors:                      3,
			expectedPerSensorBucketCapacity: 20, // 60/3 = 20
		},
		"should return minimum bucket capacity of 1 for many sensors": {
			bucketCapacity:                  5,
			numSensors:                      10,
			expectedPerSensorBucketCapacity: 1, // 5/10 = 0, but min is 1
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			limiter := mustNewLimiter(t, "test", 10, tt.bucketCapacity)
			burst := limiter.perSensorBucketCapacity(tt.numSensors)
			assert.Equal(t, tt.expectedPerSensorBucketCapacity, burst)
		})
	}
}

func TestRebalancing_DynamicSensorCount(t *testing.T) {
	// Set test deadline to fail fast if something goes wrong
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// rate=30 req/s, bucket capacity=300
	limiter := mustNewLimiter(t, "test", 30, 300)

	// Sensor 1: gets 30/1 = 30 req/s, burst = 30*10s = 300
	for range 300 {
		allowed, _ := limiter.TryConsume("sensor-1")
		assert.True(t, allowed)
	}

	// Add sensor 2: rebalances to 30/2 = 15 req/s each, burst = 15*10s = 150
	for range 150 {
		allowed, _ := limiter.TryConsume("sensor-2")
		assert.True(t, allowed)
	}

	// Add sensor 3: rebalances to 30/3 = 10 req/s each, burst = 10*10s = 100
	for range 100 {
		allowed, _ := limiter.TryConsume("sensor-3")
		assert.True(t, allowed)
	}

	// All sensors should be limited now
	allowed, _ := limiter.TryConsume("sensor-1")
	assert.False(t, allowed)
	allowed, _ = limiter.TryConsume("sensor-2")
	assert.False(t, allowed)
	allowed, _ = limiter.TryConsume("sensor-3")
	assert.False(t, allowed)

	// Wait for token refill (at 10 req/s, ~15 tokens refill in 1.5s)
	// Use select with context timeout to fail fast if test hangs
	waitDuration := 1500 * time.Millisecond
	select {
	case <-time.After(waitDuration):
		// Normal path - waited successfully
	case <-ctx.Done():
		require.Fail(t, "test deadline exceeded during sleep")
		return
	}

	// Each sensor should get ~15 tokens back - verify at least 10 work
	for range 10 {
		allowed, _ := limiter.TryConsume("sensor-1")
		assert.True(t, allowed, "sensor-1 should get tokens after refill")
		allowed, _ = limiter.TryConsume("sensor-2")
		assert.True(t, allowed, "sensor-2 should get tokens after refill")
		allowed, _ = limiter.TryConsume("sensor-3")
		assert.True(t, allowed, "sensor-3 should get tokens after refill")
	}
}

func TestOnSensorDisconnect(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 20, 100) // rate=20 req/s, bucket capacity=100

	// Create 2 sensors: each gets 20/2 = 10 req/s, burst = 10*5s = 50
	limiter.getOrCreateLimiter("sensor-1")
	limiter.getOrCreateLimiter("sensor-2")

	// Exhaust sensor-1's burst
	for i := 0; i < 50; i++ {
		allowed, _ := limiter.TryConsume("sensor-1")
		assert.True(t, allowed)
	}
	allowed, _ := limiter.TryConsume("sensor-1")
	assert.False(t, allowed, "sensor-1 should be limited after burst")

	// Disconnect sensor-2
	limiter.OnSensorDisconnect("sensor-2")

	// sensor-1 should now get full rate: 20/1 = 20 req/s, burst = 20*5s = 100
	// The existing limiter's burst is updated to 100, but tokens were exhausted
	// Just verify the sensor-2 is removed
	assert.Equal(t, 1, limiter.countActiveSensors())

	// Verify sensor-2 is no longer tracked
	_, exists := limiter.buckets.Load("sensor-2")
	assert.False(t, exists, "sensor-2 should be removed from buckets")
}

func TestOnSensorDisconnect_DisabledRateLimiter(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 0, 50)

	// Should be a no-op when rate limiting is disabled
	limiter.OnSensorDisconnect("sensor-1")
	assert.Equal(t, 0, limiter.countActiveSensors())
}

func TestOnSensorDisconnect_NonexistentSensor(t *testing.T) {
	limiter := mustNewLimiter(t, "test", 10, 50)

	// Create one sensor
	limiter.getOrCreateLimiter("sensor-1")

	// Disconnect a sensor that was never connected - should be a no-op
	limiter.OnSensorDisconnect("nonexistent-sensor")
	assert.Equal(t, 1, limiter.countActiveSensors())
}
