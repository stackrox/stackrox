package interactivesessions

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracker(t *testing.T) {
	now := time.Now()
	tracker := newTracker(time.Minute, closedSessionGracePeriod, maxSessions, func() time.Time { return now })
	exec := testEvent("audit-1", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC)
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{exec}})

	matched, found := tracker.Match("namespace", "pod", "container")
	require.True(t, found)
	assert.Equal(t, exec, matched)

	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{testEvent("audit-1", "ResponseComplete", storage.KubernetesEvent_Object_PODS_EXEC)}})

	// Sanity check: the closed session stays matchable during its grace
	// period. The close/grace expiry semantics are covered by
	// TestTrackerClosedSessionMatchesDuringGraceButNotAfter.
	matched, found = tracker.Match("namespace", "pod", "container")
	require.True(t, found)
	assert.Equal(t, exec, matched)
}

func TestTrackerNewSessionReplacesExistingSession(t *testing.T) {
	now := time.Now()
	tracker := newTracker(time.Minute, closedSessionGracePeriod, maxSessions, func() time.Time { return now })
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{
		testEvent("audit-1", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC),
	}})

	// A new session on the same container replaces the existing one. The
	// resource type (exec vs attach) is not part of the key, so an attach
	// session replaces an exec session just the same.
	attach := testEvent("audit-2", "ResponseStarted", storage.KubernetesEvent_Object_PODS_ATTACH)
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{attach}})

	matched, found := tracker.Match("namespace", "pod", "container")
	require.True(t, found)
	assert.Equal(t, attach, matched)

	// The winning session still expires after the TTL.
	now = now.Add(time.Minute)
	_, found = tracker.Match("namespace", "pod", "container")
	assert.False(t, found)
}

func TestTrackerClosedSessionMatchesDuringGraceButNotAfter(t *testing.T) {
	now := time.Now()
	tracker := newTracker(time.Minute, closedSessionGracePeriod, maxSessions, func() time.Time { return now })
	exec := testEvent("audit-1", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC)
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{exec}})

	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{
		testEvent("audit-1", "ResponseComplete", storage.KubernetesEvent_Object_PODS_EXEC),
	}})

	matched, found := tracker.Match("namespace", "pod", "container")
	require.True(t, found)
	assert.Equal(t, exec, matched)

	now = now.Add(closedSessionGracePeriod)
	_, found = tracker.Match("namespace", "pod", "container")
	assert.False(t, found)
}

func TestTrackerNewSessionReplacesClosedSessionInGrace(t *testing.T) {
	now := time.Now()
	tracker := newTracker(time.Minute, closedSessionGracePeriod, maxSessions, func() time.Time { return now })
	newSession := testEvent("audit-2", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC)
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{
		testEvent("audit-1", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC),
		testEvent("audit-1", "ResponseComplete", storage.KubernetesEvent_Object_PODS_EXEC),
		// A new session opens on the same container while the closed one is
		// still within its grace period.
		newSession,
	}})

	matched, found := tracker.Match("namespace", "pod", "container")
	require.True(t, found)
	assert.Equal(t, newSession, matched)
}

func TestTrackerCloseOfUnknownSessionIsNoOp(t *testing.T) {
	now := time.Now()
	tracker := newTracker(time.Minute, closedSessionGracePeriod, maxSessions, func() time.Time { return now })

	// Closing an unknown session neither creates nor matches a session.
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{
		testEvent("audit-1", "ResponseComplete", storage.KubernetesEvent_Object_PODS_EXEC),
	}})
	_, found := tracker.Match("namespace", "pod", "container")
	assert.False(t, found)

	// Closing an already-expired session is a no-op as well.
	exec := testEvent("audit-2", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC)
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{exec}})
	now = now.Add(time.Minute)
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{
		testEvent("audit-2", "ResponseComplete", storage.KubernetesEvent_Object_PODS_EXEC),
	}})
	_, found = tracker.Match("namespace", "pod", "container")
	assert.False(t, found)

	// Closing a session that was replaced by a newer one on the same
	// container is a no-op as well.
	replacing := testEvent("audit-4", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC)
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{
		testEvent("audit-3", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC),
	}})
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{replacing}})
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{
		testEvent("audit-3", "ResponseComplete", storage.KubernetesEvent_Object_PODS_EXEC),
	}})
	matched, found := tracker.Match("namespace", "pod", "container")
	require.True(t, found)
	assert.Equal(t, replacing, matched)
}

func TestTrackerEvictsOldestSessionAtCap(t *testing.T) {
	now := time.Now()
	tracker := newTracker(time.Hour, closedSessionGracePeriod, 2, func() time.Time { return now })

	// Sessions on distinct keys are tracked independently, so use different
	// pod names to build up separate keys.
	podEvent := func(auditID, pod, stage string) *storage.KubernetesEvent {
		event := testEvent(auditID, stage, storage.KubernetesEvent_Object_PODS_EXEC)
		event.Object.Name = pod
		return event
	}

	// The cap is reached with sessions on two distinct keys.
	first := podEvent("audit-1", "pod-1", "ResponseStarted")
	second := podEvent("audit-2", "pod-2", "ResponseStarted")
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{first, second}})

	// A third session on a new key evicts the oldest session to make room.
	third := podEvent("audit-3", "pod-3", "ResponseStarted")
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{third}})

	_, found := tracker.Match("namespace", "pod-1", "container")
	assert.False(t, found)
	for pod, want := range map[string]*storage.KubernetesEvent{
		"pod-2": second,
		"pod-3": third,
	} {
		matched, found := tracker.Match("namespace", pod, "container")
		require.True(t, found)
		assert.Equal(t, want, matched)
	}

	// Preference: with a closed (grace-expiring) session and two active ones
	// at the cap, the closed/earliest-expiring session is the one evicted.
	tracker = newTracker(time.Hour, closedSessionGracePeriod, 3, func() time.Time { return now })
	closed := podEvent("audit-4", "pod-4", "ResponseStarted")
	activeFive := podEvent("audit-5", "pod-5", "ResponseStarted")
	activeSix := podEvent("audit-6", "pod-6", "ResponseStarted")
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{
		closed,
		podEvent("audit-4", "pod-4", "ResponseComplete"),
		activeFive,
		activeSix,
	}})
	newest := podEvent("audit-7", "pod-7", "ResponseStarted")
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{newest}})

	_, found = tracker.Match("namespace", "pod-4", "container")
	assert.False(t, found)
	for pod, want := range map[string]*storage.KubernetesEvent{
		"pod-5": activeFive,
		"pod-6": activeSix,
		"pod-7": newest,
	} {
		matched, found := tracker.Match("namespace", pod, "container")
		require.True(t, found)
		assert.Equal(t, want, matched)
	}
}

func TestTrackerDuplicateResponseStartedResetsClosedSession(t *testing.T) {
	now := time.Now()
	tracker := newTracker(time.Minute, closedSessionGracePeriod, maxSessions, func() time.Time { return now })
	exec := testEvent("audit-1", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC)
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{exec}})
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{
		testEvent("audit-1", "ResponseComplete", storage.KubernetesEvent_Object_PODS_EXEC),
	}})

	// A duplicate ResponseStarted (e.g. an audit-log replay) resets the
	// closed session to active with a fresh TTL.
	replayed := testEvent("audit-1", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC)
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{replayed}})

	// Past the grace period the closed session would have expired, but well
	// before the TTL of the reset session: it must still match.
	now = now.Add(closedSessionGracePeriod + time.Second)
	matched, found := tracker.Match("namespace", "pod", "container")
	require.True(t, found)
	assert.Equal(t, replayed, matched)

	now = now.Add(time.Minute)
	_, found = tracker.Match("namespace", "pod", "container")
	assert.False(t, found)
}

func testEvent(auditID, stage string, resource storage.KubernetesEvent_Object_Resource) *storage.KubernetesEvent {
	event := &storage.KubernetesEvent{
		Id:         auditID,
		Object:     &storage.KubernetesEvent_Object{Name: "pod", Namespace: "namespace", Resource: resource},
		AuditStage: stage,
	}
	if resource == storage.KubernetesEvent_Object_PODS_EXEC {
		event.ObjectArgs = &storage.KubernetesEvent_PodExecArgs_{PodExecArgs: &storage.KubernetesEvent_PodExecArgs{Container: "container"}}
	} else {
		event.ObjectArgs = &storage.KubernetesEvent_PodAttachArgs_{PodAttachArgs: &storage.KubernetesEvent_PodAttachArgs{Container: "container"}}
	}
	return event
}
