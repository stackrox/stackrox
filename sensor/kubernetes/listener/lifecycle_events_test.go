package listener

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

func TestSoftRestartEvent_IsExpired(t *testing.T) {
	t.Run("nil validity is not expired", func(t *testing.T) {
		e := &SoftRestartEvent{}
		assert.False(t, e.IsExpired())
	})
	t.Run("active context is not expired", func(t *testing.T) {
		e := &SoftRestartEvent{Validity: context.Background()}
		assert.False(t, e.IsExpired())
	})
	t.Run("cancelled context is expired", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		e := &SoftRestartEvent{Validity: ctx}
		assert.True(t, e.IsExpired())
	})
}

func TestResourceSyncFinishedEvent_IsExpired(t *testing.T) {
	t.Run("nil validity is not expired", func(t *testing.T) {
		e := &ResourceSyncFinishedEvent{}
		assert.False(t, e.IsExpired())
	})
	t.Run("active context is not expired", func(t *testing.T) {
		e := &ResourceSyncFinishedEvent{Validity: context.Background()}
		assert.False(t, e.IsExpired())
	})
	t.Run("cancelled context is expired", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		e := &ResourceSyncFinishedEvent{Validity: ctx}
		assert.True(t, e.IsExpired())
	})
}

func TestSoftRestartEvent_String(t *testing.T) {
	e := &SoftRestartEvent{Text: "CRD resources changed"}
	assert.Equal(t, "CRD resources changed", e.String())
}
