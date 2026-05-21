package propagation

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoller_DetectsChange(t *testing.T) {
	var currentVersion int64
	var mu sync.Mutex

	// Mock fetcher that returns the current version
	fetch := func() (int64, error) {
		mu.Lock()
		defer mu.Unlock()
		return currentVersion, nil
	}

	// Track onChange calls
	var changes []struct{ old, new int64 }
	var changeMu sync.Mutex
	onChange := func(old, new int64) {
		changeMu.Lock()
		defer changeMu.Unlock()
		changes = append(changes, struct{ old, new int64 }{old, new})
	}

	// Create poller with 10ms interval for fast testing
	poller := NewPoller(fetch, onChange, 10*time.Millisecond)
	poller.Start()

	// Wait for initial poll
	time.Sleep(20 * time.Millisecond)

	// Change version: 0 → 1
	mu.Lock()
	currentVersion = 1
	mu.Unlock()

	// Wait for poller to detect change
	require.Eventually(t, func() bool {
		changeMu.Lock()
		defer changeMu.Unlock()
		return len(changes) > 0
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Verify onChange was called with correct values
	changeMu.Lock()
	assert.Equal(t, int64(0), changes[0].old)
	assert.Equal(t, int64(1), changes[0].new)
	changeMu.Unlock()

	poller.Stop()
}

func TestPoller_IgnoresUnchangedVersion(t *testing.T) {
	// Start at version 0, which matches the initial atomic.Int64 value
	fetch := func() (int64, error) {
		return 0, nil
	}

	var changeCount int
	var mu sync.Mutex
	onChange := func(old, new int64) {
		mu.Lock()
		defer mu.Unlock()
		changeCount++
	}

	poller := NewPoller(fetch, onChange, 10*time.Millisecond)
	poller.Start()

	// Wait for several polls
	time.Sleep(50 * time.Millisecond)

	poller.Stop()

	// onChange should never be called since version doesn't change
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 0, changeCount)
}
