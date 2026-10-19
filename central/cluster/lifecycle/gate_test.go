package lifecycle

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGateBlocksDeletionUntilActiveWriteCompletes(t *testing.T) {
	clusterID := "cluster-id"

	gate := New()
	writeStarted := make(chan struct{})
	releaseWrite := make(chan struct{})
	writeFinished := make(chan struct{})
	go func() {
		active, release := gate.Enter(clusterID)
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

	deletionStarted := make(chan func(), 1)
	go func() { deletionStarted <- gate.BeginDeletion(clusterID) }()
	require.Eventually(t, func() bool {
		active, release := gate.Enter(clusterID)
		if release != nil {
			release()
		}

		return !active
	}, time.Second, 5*time.Millisecond)

	// yield -> to give additional cycles to BeginDeletion.
	runtime.Gosched()
	require.Empty(t, deletionStarted)

	close(releaseWrite)
	<-writeFinished
	releaseDeletion := <-deletionStarted
	releaseDeletion()

	active, release := gate.Enter(clusterID)
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
