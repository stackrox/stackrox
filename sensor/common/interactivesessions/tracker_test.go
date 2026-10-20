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
	tracker := newTracker(time.Minute, func() time.Time { return now })
	exec := testEvent("audit-1", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC)
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{exec}})

	matched, found := tracker.Match("namespace", "pod", "container")
	require.True(t, found)
	assert.Equal(t, exec, matched)

	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{testEvent("audit-1", "ResponseComplete", storage.KubernetesEvent_Object_PODS_EXEC)}})
	_, found = tracker.Match("namespace", "pod", "container")
	assert.False(t, found)
}

func TestTrackerDoesNotAttributeAmbiguousOrExpiredSessions(t *testing.T) {
	now := time.Now()
	tracker := newTracker(time.Minute, func() time.Time { return now })
	tracker.Observe(&sensor.AuditEvents{Events: []*storage.KubernetesEvent{
		testEvent("audit-1", "ResponseStarted", storage.KubernetesEvent_Object_PODS_EXEC),
		testEvent("audit-2", "ResponseStarted", storage.KubernetesEvent_Object_PODS_ATTACH),
	}})

	_, found := tracker.Match("namespace", "pod", "container")
	assert.False(t, found)

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
