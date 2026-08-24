package centralbound

import (
	"context"
	"testing"

	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
)

func TestCentralBoundEvent_TopicAndLane(t *testing.T) {
	e := &CentralBoundEvent{Msg: message.New(nil)}
	assert.Equal(t, pubsub.CentralBoundTopic, e.Topic())
	assert.Equal(t, pubsub.CentralBoundLane, e.Lane())
}

func TestCentralBoundEvent_IsExpired(t *testing.T) {
	t.Run("nil context is not expired", func(t *testing.T) {
		e := &CentralBoundEvent{Msg: &message.ExpiringMessage{}}
		assert.False(t, e.IsExpired())
	})
	t.Run("active context is not expired", func(t *testing.T) {
		e := &CentralBoundEvent{Msg: message.New(nil)}
		assert.False(t, e.IsExpired())
	})
	t.Run("cancelled context is expired", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		e := &CentralBoundEvent{Msg: message.NewExpiring(ctx, nil)}
		assert.True(t, e.IsExpired())
	})
}
