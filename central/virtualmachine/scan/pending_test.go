package scan

import (
	"testing"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func TestOverlappingScans(t *testing.T) {
	tracker := &Tracker{}
	finishFirst := tracker.Start("vm-1")
	finishSecond := tracker.Start("vm-1")
	finishOther := tracker.Start("vm-2")
	assert.True(t, tracker.IsPending("vm-1"))
	assert.False(t, tracker.IsPending("vm-3"))

	finishFirst()
	finishFirst()
	assert.True(t, tracker.IsPending("vm-1"))
	finishSecond()
	assert.False(t, tracker.IsPending("vm-1"))
	assert.True(t, tracker.IsPending("vm-2"))
	finishOther()
	assert.Empty(t, tracker.counts, "completed scans must not retain VM IDs")
}

func TestConcurrentScans(t *testing.T) {
	tracker := &Tracker{}
	var workers sync.WaitGroup
	for range 100 {
		workers.Go(func() {
			finish := tracker.Start("vm-1")
			defer finish()
			assert.True(t, tracker.IsPending("vm-1"))
		})
	}
	workers.Wait()
	assert.False(t, tracker.IsPending("vm-1"))
	assert.Empty(t, tracker.counts)
}
