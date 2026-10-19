package saml

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// testClock provides a controllable clock for tests
type testClock struct {
	now atomic.Value // *time.Time
}

func newTestClock(initial time.Time) *testClock {
	tc := &testClock{}
	tc.now.Store(&initial)
	return tc
}

func (tc *testClock) Now() time.Time {
	return *tc.now.Load().(*time.Time)
}

func (tc *testClock) SetTime(t time.Time) {
	tc.now.Store(&t)
}

func TestReplayCache_Add(t *testing.T) {
	cache := newReplayCache()
	defer cache.stop()

	assertionID := "assertion-id-1"
	expiresAt := time.Now().Add(1 * time.Hour)

	// First call should return true (not seen before)
	seen := cache.add(assertionID, expiresAt)
	assert.True(t, seen, "first call to add should return false")

	// Second call with same ID should return false (replay detected)
	seen = cache.add(assertionID, expiresAt)
	assert.False(t, seen, "second call to add with same ID should return true")

	// Third call should still return false
	seen = cache.add(assertionID, expiresAt)
	assert.False(t, seen, "third call to add with same ID should return true")
}

func TestReplayCache_DifferentAssertionIDs(t *testing.T) {
	cache := newReplayCache()
	defer cache.stop()

	expiresAt := time.Now().Add(1 * time.Hour)

	// Different assertion IDs should all be new
	assert.True(t, cache.add("assertion-1", expiresAt))
	assert.True(t, cache.add("assertion-2", expiresAt))
	assert.True(t, cache.add("assertion-3", expiresAt))

	// But replaying the same IDs should be detected
	assert.False(t, cache.add("assertion-1", expiresAt))
	assert.False(t, cache.add("assertion-2", expiresAt))
	assert.False(t, cache.add("assertion-3", expiresAt))
}

func TestReplayCache_Prune(t *testing.T) {
	clock := newTestClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	cache := newReplayCacheWithClock(clock.Now)
	defer cache.stop()

	now := clock.Now()

	// Add entries with different expiration times
	shortExpiry1 := now.Add(10 * time.Minute)
	shortExpiry2 := now.Add(20 * time.Minute)
	cache.add("will-expire-1", shortExpiry1)
	cache.add("will-expire-2", shortExpiry2)
	cache.add("active-1", now.Add(1*time.Hour)) // Still active
	cache.add("active-2", now.Add(2*time.Hour)) // Still active

	// Verify all entries are initially present
	cache.mu.RLock()
	initialCount := len(cache.entries)
	cache.mu.RUnlock()
	assert.Equal(t, 4, initialCount, "all 4 entries should be in cache initially")

	// Advance the clock past the short-lived entries' expiration
	clock.SetTime(now.Add(30 * time.Minute))

	// Run pruning
	cache.prune()

	// After pruning, only active entries should remain
	cache.mu.RLock()
	afterPruningCount := len(cache.entries)
	_, hasExpired1 := cache.entries["will-expire-1"]
	_, hasExpired2 := cache.entries["will-expire-2"]
	_, hasActive1 := cache.entries["active-1"]
	_, hasActive2 := cache.entries["active-2"]
	cache.mu.RUnlock()

	assert.Equal(t, 2, afterPruningCount, "only 2 active entries should remain after pruning")
	assert.False(t, hasExpired1, "will-expire-1 should be removed")
	assert.False(t, hasExpired2, "will-expire-2 should be removed")
	assert.True(t, hasActive1, "active-1 should remain")
	assert.True(t, hasActive2, "active-2 should remain")
}

func TestReplayCache_PruningAutomatic(t *testing.T) {
	// This test verifies that the automatic pruning goroutine can be started and stopped.
	clock := newTestClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	cache := newReplayCacheWithClock(clock.Now)

	// Start pruning
	cache.start()

	now := clock.Now()
	cache.add("test-id", now.Add(1*time.Hour))

	// Stop the cache to halt pruning goroutine
	cache.stop()

	// Verify we can still use the cache after stop
	// (stop only stops the pruning goroutine, not the cache itself)
	assert.False(t, cache.add("test-id", now.Add(1*time.Hour)))

	// Verify we can restart pruning
	cache.start()

	// Cache should still work with pruning running again
	assert.False(t, cache.add("test-id", now.Add(1*time.Hour)))

	cache.stop()
}

func TestReplayCache_Start(t *testing.T) {
	cache := newReplayCache()
	defer cache.stop()

	// Start should be safe to call multiple times
	cache.start()
	cache.start()
	cache.start()

	// Cache should be functional with pruning running
	expiresAt := time.Now().Add(1 * time.Hour)
	assert.True(t, cache.add("test-id", expiresAt))
	assert.False(t, cache.add("test-id", expiresAt))
}

func TestReplayCache_Stop(t *testing.T) {
	cache := newReplayCache()

	// Stop should be safe to call multiple times
	cache.stop()
	cache.stop()
	cache.stop()

	// Cache should still be functional after stop
	expiresAt := time.Now().Add(1 * time.Hour)
	assert.True(t, cache.add("test-id", expiresAt))
	assert.False(t, cache.add("test-id", expiresAt))
}

func TestReplayCache_ConcurrentAccess(t *testing.T) {
	cache := newReplayCache()
	defer cache.stop()

	expiresAt := time.Now().Add(1 * time.Hour)
	assertionID := "concurrent-test-id"

	// Simulate concurrent access from multiple goroutines
	const numGoroutines = 10
	results := make(chan bool, numGoroutines)

	for range numGoroutines {
		go func() {
			results <- cache.add(assertionID, expiresAt)
		}()
	}

	// Collect results
	seenCount := 0
	newCount := 0
	for range numGoroutines {
		if <-results {
			newCount++
		} else {
			seenCount++
		}
	}

	// Exactly one goroutine should have succeeded in marking it as new
	assert.Equal(t, 1, newCount, "exactly one goroutine should mark the ID as new")
	assert.Equal(t, numGoroutines-1, seenCount, "all other goroutines should detect it as seen")
}

func TestReplayCache_EmptyAssertionID(t *testing.T) {
	cache := newReplayCache()
	defer cache.stop()

	expiresAt := time.Now().Add(1 * time.Hour)

	// Empty assertion IDs should be tracked like any other ID
	assert.True(t, cache.add("", expiresAt))
	assert.False(t, cache.add("", expiresAt))
}
