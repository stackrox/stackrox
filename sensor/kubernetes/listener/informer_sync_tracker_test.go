package listener

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
)

// newTestTracker creates a tracker with a very long interval so the background goroutine
// never fires during tests. Tests call getState() directly to verify behavior.
// Cleanups run in LIFO order: stop() runs after close(stopC).
func newTestTracker(t *testing.T) *informerSyncTracker {
	t.Helper()
	stopC := make(chan struct{})

	tracker := newInformerSyncTracker(24*time.Hour, stopC)
	t.Cleanup(func() { tracker.stop() })
	t.Cleanup(func() { close(stopC) })
	return tracker
}

func pendingNames(t *testing.T, state syncState) []string {
	t.Helper()
	names := make([]string, 0, len(state.pending))
	for _, p := range state.pending {
		names = append(names, p.name)
	}
	return names
}

func TestInformerSyncTracker_AllSynced(t *testing.T) {
	tracker := newTestTracker(t)

	tracker.register(informerNamespaces)
	tracker.register(informerSecrets)
	tracker.register(informerPods)

	tracker.markSynced(informerNamespaces)
	tracker.markSynced(informerSecrets)
	tracker.markSynced(informerPods)

	state := tracker.getState()
	assert.Len(t, state.synced, 3)
	assert.Empty(t, state.pending)
}

func TestInformerSyncTracker_SomeStuck(t *testing.T) {
	tracker := newTestTracker(t)

	tracker.register(informerNamespaces)
	tracker.register(informerSecrets)
	tracker.register(informerNetworkPolicies)
	tracker.register(informerNodes)

	tracker.markSynced(informerNamespaces)
	tracker.markSynced(informerSecrets)

	state := tracker.getState()
	assert.Len(t, state.synced, 2)
	assert.Len(t, state.pending, 2)
	assert.Contains(t, pendingNames(t, state), informerNetworkPolicies)
	assert.Contains(t, pendingNames(t, state), informerNodes)
	for _, p := range state.pending {
		assert.GreaterOrEqual(t, p.pending, time.Duration(0))
	}
}

func TestInformerSyncTracker_NoneRegistered(t *testing.T) {
	tracker := newTestTracker(t)

	state := tracker.getState()
	assert.Empty(t, state.synced)
	assert.Empty(t, state.pending)
}

func TestInformerSyncTracker_NilSafe(t *testing.T) {
	var tracker *informerSyncTracker

	tracker.register(informerNamespaces)
	tracker.markSynced(informerNamespaces)
	tracker.stop()
}

func TestInformerSyncTracker_MarkSyncedUnknown(t *testing.T) {
	tracker := newTestTracker(t)

	tracker.register(informerNamespaces)
	tracker.markSynced("UnknownInformer_test")

	state := tracker.getState()
	assert.Empty(t, state.synced)
	assert.Len(t, state.pending, 1)
	assert.Equal(t, informerNamespaces, state.pending[0].name)
}

func TestInformerSyncTracker_MarkSyncedIdempotent(t *testing.T) {
	tracker := newTestTracker(t)

	tracker.register(informerNamespaces)
	tracker.markSynced(informerNamespaces)

	tracker.mu.Lock()
	firstSyncTime := tracker.informers[informerNamespaces].syncedAt
	tracker.mu.Unlock()

	tracker.markSynced(informerNamespaces)

	tracker.mu.Lock()
	assert.Equal(t, firstSyncTime, tracker.informers[informerNamespaces].syncedAt, "sync time should not change")
	tracker.mu.Unlock()
}

func TestInformerSyncTracker_StopSignal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stopC := make(chan struct{})
		tracker := newInformerSyncTracker(10*time.Second, stopC)

		tracker.register(informerNamespaces)

		close(stopC)
		synctest.Wait()

		done := make(chan struct{})
		go func() {
			tracker.stop()
			close(done)
		}()
		synctest.Wait()

		select {
		case <-done:
		default:
			t.Fatal("Tracker goroutine did not exit within timeout")
		}
	})
}

func TestInformerSyncTracker_ProgressTracking(t *testing.T) {
	tracker := newTestTracker(t)

	tracker.register(informerNamespaces)
	tracker.register(informerSecrets)
	tracker.register(informerRoles)
	tracker.register(informerNetworkPolicies)
	tracker.register(informerDeployments)

	tracker.markSynced(informerNamespaces)
	tracker.markSynced(informerSecrets)
	tracker.markSynced(informerRoles)

	state := tracker.getState()
	assert.Len(t, state.synced, 3)
	assert.Len(t, state.pending, 2)
	assert.Contains(t, pendingNames(t, state), informerNetworkPolicies)
	assert.Contains(t, pendingNames(t, state), informerDeployments)
}

func TestInformerSyncTracker_DuplicateRegistration(t *testing.T) {
	tracker := newTestTracker(t)

	tracker.register(informerNamespaces)
	tracker.register(informerNamespaces)

	state := tracker.getState()
	assert.Len(t, state.pending, 1, "duplicate registration should be ignored")
}

func TestSyncState_Log_AllSynced(t *testing.T) {
	state := syncState{
		synced:  []string{informerNamespaces, informerSecrets},
		pending: nil,
	}
	assert.NotPanics(t, func() {
		state.log()
	})
}

func TestSyncState_Log_Pending(t *testing.T) {
	state := syncState{
		synced: []string{informerNamespaces},
		pending: []pendingInformer{
			{name: informerNetworkPolicies, pending: 2*time.Minute + 30*time.Second},
			{name: informerNodes, pending: 2*time.Minute + 30*time.Second},
		},
	}
	assert.NotPanics(t, func() {
		state.log()
	})
}

func TestSyncState_Log_Empty(t *testing.T) {
	state := syncState{}
	assert.NotPanics(t, func() {
		state.log()
	})
}

func TestInformerSyncTracker_RunExitsWhenAllSynced(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stopC := make(chan struct{})
		tracker := newInformerSyncTracker(30*time.Second, stopC)

		tracker.register(informerNamespaces)
		tracker.register(informerSecrets)

		// Advance past the first tick -- should report pending informers.
		time.Sleep(30 * time.Second)
		synctest.Wait()

		state := tracker.getState()
		assert.Len(t, state.pending, 2, "both informers should be pending after first tick")

		// Sync one informer, advance to second tick.
		tracker.markSynced(informerNamespaces)
		time.Sleep(30 * time.Second)
		synctest.Wait()

		state = tracker.getState()
		assert.Len(t, state.synced, 1)
		assert.Len(t, state.pending, 1)

		// Sync the last informer, advance to third tick.
		// run() should exit after this tick since all are synced.
		tracker.markSynced(informerSecrets)
		time.Sleep(30 * time.Second)
		synctest.Wait()

		// The goroutine should have exited -- stop() should return immediately.
		done := make(chan struct{})
		go func() {
			tracker.stop()
			close(done)
		}()
		synctest.Wait()

		select {
		case <-done:
		default:
			t.Fatal("run() goroutine did not exit after all informers synced")
		}
	})
}

func TestInformerSyncTracker_RunReportsOnEachTick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stopC := make(chan struct{})
		tracker := newInformerSyncTracker(10*time.Second, stopC)

		tracker.register(informerNetworkPolicies)

		// First tick at 10s -- should be pending.
		time.Sleep(10 * time.Second)
		synctest.Wait()

		state := tracker.getState()
		assert.Len(t, state.pending, 1)
		assert.Equal(t, informerNetworkPolicies, state.pending[0].name)
		assert.Equal(t, 10*time.Second, state.pending[0].pending)

		// Second tick at 20s -- still pending, duration should increase.
		time.Sleep(10 * time.Second)
		synctest.Wait()

		state = tracker.getState()
		assert.Len(t, state.pending, 1)
		assert.Equal(t, 20*time.Second, state.pending[0].pending)

		// Cleanup
		close(stopC)
		synctest.Wait()
		tracker.stop()
	})
}

func TestInformerSyncTracker_RunStopsOnStopC(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stopC := make(chan struct{})
		tracker := newInformerSyncTracker(30*time.Second, stopC)

		tracker.register(informerNamespaces)

		close(stopC)
		synctest.Wait()

		done := make(chan struct{})
		go func() {
			tracker.stop()
			close(done)
		}()
		synctest.Wait()

		select {
		case <-done:
		default:
			t.Fatal("run() goroutine did not exit after stopC closed")
		}
	})
}

func TestInformerSyncTracker_RunExitsOnNoRegistrationsTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stopC := make(chan struct{})
		tracker := newInformerSyncTracker(30*time.Second, stopC)

		// Don't register any informers. After noRegistrationsTimeout (60s)
		// the tracker should exit on its own.
		time.Sleep(noRegistrationsTimeout)
		synctest.Wait()

		done := make(chan struct{})
		go func() {
			tracker.stop()
			close(done)
		}()
		synctest.Wait()

		select {
		case <-done:
		default:
			t.Fatal("run() goroutine did not exit after no-registrations timeout")
		}
	})
}

func TestInformerSyncTracker_RunIgnoresNoRegistrationsTimeoutAfterRegister(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stopC := make(chan struct{})
		tracker := newInformerSyncTracker(30*time.Second, stopC)

		tracker.register(informerNamespaces)

		// Advance past the no-registrations timeout. The tracker should NOT
		// exit because an informer was registered (still pending).
		time.Sleep(noRegistrationsTimeout)
		synctest.Wait()

		state := tracker.getState()
		assert.Len(t, state.pending, 1, "informer should still be pending")

		// Sync the informer and advance to next tick so run() exits normally.
		tracker.markSynced(informerNamespaces)
		time.Sleep(30 * time.Second)
		synctest.Wait()

		done := make(chan struct{})
		go func() {
			tracker.stop()
			close(done)
		}()
		synctest.Wait()

		select {
		case <-done:
		default:
			t.Fatal("run() goroutine did not exit after all informers synced")
		}
	})
}

func TestInformerSyncTracker_FeatureFlagDisabled(t *testing.T) {
	t.Setenv(features.SensorInformerWatchdog.EnvVar(), "false")
	assert.False(t, features.SensorInformerWatchdog.Enabled())
}

func TestInformerSyncTracker_FeatureFlagEnabled(t *testing.T) {
	t.Setenv(features.SensorInformerWatchdog.EnvVar(), "true")
	assert.True(t, features.SensorInformerWatchdog.Enabled())
}
