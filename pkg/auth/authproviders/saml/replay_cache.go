package saml

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// Clock is a function that returns the current time. Used for testing.
type Clock func() time.Time

// replayCache tracks SAML assertion IDs to prevent replay attacks.
// It automatically expires entries at their SessionNotOnOrAfter time.
type replayCache struct {
	// mu guards access to entries
	mu      sync.RWMutex
	entries map[string]time.Time // assertion ID -> expiration time

	// cleanupMu guards the start and stop of cache pruning
	cleanupMu sync.Mutex
	stopSig   concurrency.Signal
	isRunning bool // Tracks whether pruning goroutine is active

	clock Clock // Clock function for obtaining current time
}

func newReplayCache() *replayCache {
	return newReplayCacheWithClock(time.Now)
}

func newReplayCacheWithClock(clock Clock) *replayCache {
	if clock == nil {
		clock = time.Now
	}
	return &replayCache{
		entries: make(map[string]time.Time),
		clock:   clock,
	}
}

// add checks if an assertion ID has been seen before and is still valid.
// If not seen or expired, it records the assertion ID with the given expiration time.
// Returns false if the assertion was already seen and not expired (replay attack), true otherwise.
func (c *replayCache) add(assertionID string, expiredAt time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[assertionID]; exists {
		return false // Already seen - likely a replay attack
	}

	c.entries[assertionID] = expiredAt
	return true
}

// prune removes expired entries from the cache.
func (c *replayCache) prune() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.clock()
	for id, expiredAt := range c.entries {
		if now.After(expiredAt) {
			delete(c.entries, id)
		}
	}
}

// start starts periodic pruning of expired entries.
// Can be called multiple times safely - it will start the pruning goroutine if not already running.
func (c *replayCache) start() {
	c.cleanupMu.Lock()
	defer c.cleanupMu.Unlock()

	// If pruning is already running, do nothing
	if c.isRunning {
		return
	}

	c.stopSig = concurrency.NewSignal()
	c.stopSig.Reset()
	c.isRunning = true

	go func(sigDone <-chan struct{}) {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.prune()
			case <-sigDone:
				return
			}
		}
	}(c.stopSig.Done())
}

// stop halts the pruning goroutine. Can be called multiple times safely.
func (c *replayCache) stop() {
	c.cleanupMu.Lock()
	defer c.cleanupMu.Unlock()
	c.stopSig.Signal()
	c.isRunning = false
}
