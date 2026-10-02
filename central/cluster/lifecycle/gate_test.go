package lifecycle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGateBlocksDeletionUntilActiveWriteCompletes(t *testing.T) {
	gate := New()
	writeStarted := make(chan struct{})
	releaseWrite := make(chan struct{})
	writeFinished := make(chan struct{})
	go func() {
		active, release := gate.Enter("cluster-id")
		if !active {
			t.Error("active write was rejected")
			return
		}
		defer release()
		close(writeStarted)
		<-releaseWrite
		close(writeFinished)
	}()
	<-writeStarted

	deletionStarted := make(chan func())
	go func() { deletionStarted <- gate.BeginDeletion("cluster-id") }()
	require.Never(t, func() bool {
		select {
		case <-deletionStarted:
			return true
		default:
			return false
		}
	}, 20*time.Millisecond, time.Millisecond)

	close(releaseWrite)
	<-writeFinished
	releaseDeletion := <-deletionStarted
	releaseDeletion()

	active, release := gate.Enter("cluster-id")
	require.True(t, active)
	release()
}

// TestBeginDeletionDropsWriteLockBeforeCleanup checks that BeginDeletion only
// holds the write lock long enough to drain in-flight leases. The dev mutex
// watchdog aborts the process if that lock stays held through cleanup.
func TestBeginDeletionDropsWriteLockBeforeCleanup(t *testing.T) {
	gate := New()
	releaseDeletion := gate.BeginDeletion("cluster-id")
	t.Cleanup(releaseDeletion)

	e := gate.entries["cluster-id"]
	require.NotNil(t, e)
	require.True(t, e.mu.TryLock(), "write lock still held after BeginDeletion returned")
	e.mu.Unlock()

	active, release := gate.Enter("cluster-id")
	require.False(t, active)
	require.Nil(t, release)
}
