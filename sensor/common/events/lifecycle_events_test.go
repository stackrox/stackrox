package events

import (
	"context"
	"testing"

	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
)

func TestSoftRestartEvent_TopicAndLane(t *testing.T) {
	e := &SoftRestartEvent{}
	assert.Equal(t, pubsub.SoftRestartTopic, e.Topic())
	assert.Equal(t, pubsub.SoftRestartLane, e.Lane())
}

func TestResourceSyncFinishedEvent_TopicAndLane(t *testing.T) {
	e := &ResourceSyncFinishedEvent{}
	assert.Equal(t, pubsub.ResourceSyncFinishedTopic, e.Topic())
	assert.Equal(t, pubsub.ResourceSyncFinishedLane, e.Lane())
}

// TestLifecycleEvent_IsExpired verifies the shared IsExpired behavior for both
// SoftRestartEvent and ResourceSyncFinishedEvent.
func TestLifecycleEvent_IsExpired(t *testing.T) {
	tests := map[string]struct {
		event    interface{ IsExpired() bool }
		expected bool
	}{
		"SoftRestart with nil validity": {
			event:    &SoftRestartEvent{},
			expected: false,
		},
		"SoftRestart with active context": {
			event:    &SoftRestartEvent{LifecycleEvent{Validity: context.Background()}},
			expected: false,
		},
		"SoftRestart with cancelled context": {
			event: func() *SoftRestartEvent {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return &SoftRestartEvent{LifecycleEvent{Validity: ctx}}
			}(),
			expected: true,
		},
		"ResourceSyncFinished with nil validity": {
			event:    &ResourceSyncFinishedEvent{},
			expected: false,
		},
		"ResourceSyncFinished with active context": {
			event:    &ResourceSyncFinishedEvent{LifecycleEvent{Validity: context.Background()}},
			expected: false,
		},
		"ResourceSyncFinished with cancelled context": {
			event: func() *ResourceSyncFinishedEvent {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return &ResourceSyncFinishedEvent{LifecycleEvent{Validity: ctx}}
			}(),
			expected: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.event.IsExpired())
		})
	}
}

func TestSoftRestartEvent_String(t *testing.T) {
	e := &SoftRestartEvent{LifecycleEvent{Text: "CRD resources changed"}}
	assert.Equal(t, "CRD resources changed", e.String())
}
