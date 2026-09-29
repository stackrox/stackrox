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
