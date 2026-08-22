package centralbound

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingDispatcher invokes the registered callback synchronously in the
// caller's goroutine (no lane/dispatcher goroutine involved), so tests using
// it can read from ResponsesC with a non-blocking `default` case instead of
// racing a real timeout.
type capturingDispatcher struct {
	consumerID pubsub.ConsumerID
	topic      pubsub.Topic
	laneID     pubsub.LaneID
	callback   pubsub.EventCallback
}

func (c *capturingDispatcher) RegisterConsumer(_ pubsub.ConsumerID, _ pubsub.Topic, _ pubsub.EventCallback) error {
	return nil
}

func (c *capturingDispatcher) RegisterConsumerToLane(id pubsub.ConsumerID, t pubsub.Topic, l pubsub.LaneID, cb pubsub.EventCallback) error {
	c.consumerID = id
	c.topic = t
	c.laneID = l
	c.callback = cb
	return nil
}

func (c *capturingDispatcher) Publish(_ pubsub.Event) error { return nil }
func (c *capturingDispatcher) Stop()                        {}

func TestBridge_RegistrationWiring(t *testing.T) {
	capturing := &capturingDispatcher{}
	b, err := NewBridge(capturing)
	require.NoError(t, err)
	require.NotNil(t, b)

	assert.Equal(t, pubsub.CentralBoundBridgeConsumer, capturing.consumerID)
	assert.Equal(t, pubsub.CentralBoundTopic, capturing.topic)
	assert.Equal(t, pubsub.CentralBoundLane, capturing.laneID)
	require.NotNil(t, capturing.callback)
}

func TestBridge_ReceivesEvent(t *testing.T) {
	capturing := &capturingDispatcher{}
	b, err := NewBridge(capturing)
	require.NoError(t, err)

	msg := message.New(&central.MsgFromSensor{})
	require.NoError(t, capturing.callback(&CentralBoundEvent{Msg: msg}))

	select {
	case received := <-b.ResponsesC():
		assert.Same(t, msg, received)
	default:
		t.Fatal("expected message on ResponsesC")
	}
}

func TestBridge_SkipsExpiredEvent(t *testing.T) {
	capturing := &capturingDispatcher{}
	b, err := NewBridge(capturing)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, capturing.callback(&CentralBoundEvent{
		Msg: message.NewExpiring(ctx, &central.MsgFromSensor{}),
	}))

	select {
	case msg := <-b.ResponsesC():
		t.Fatalf("expected no message, got %v", msg)
	default:
	}
}

func TestBridge_RejectsWrongEventType(t *testing.T) {
	capturing := &capturingDispatcher{}
	_, err := NewBridge(capturing)
	require.NoError(t, err)

	err = capturing.callback(&wrongEvent{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

type wrongEvent struct{}

func (w *wrongEvent) Topic() pubsub.Topic { return pubsub.CentralBoundTopic }
func (w *wrongEvent) Lane() pubsub.LaneID { return pubsub.CentralBoundLane }

func TestBridge_HandleEventAfterStopDoesNotPanic(t *testing.T) {
	capturing := &capturingDispatcher{}
	b, err := NewBridge(capturing)
	require.NoError(t, err)

	b.Stop()

	assert.NotPanics(t, func() {
		_ = capturing.callback(&CentralBoundEvent{
			Msg: message.New(&central.MsgFromSensor{}),
		})
	})
}

func TestBridge_ConcurrentPublish(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		capturing := &capturingDispatcher{}
		b, err := NewBridge(capturing)
		require.NoError(t, err)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)
		for range goroutines {
			go func() {
				defer wg.Done()
				_ = capturing.callback(&CentralBoundEvent{
					Msg: message.New(&central.MsgFromSensor{}),
				})
			}()
		}
		wg.Wait()
		synctest.Wait()

		// defaultBufferSize (100) comfortably fits all 50 sends, so every message is
		// already buffered on ResponsesC by now; drain it from this single goroutine.
		received := 0
		for {
			select {
			case <-b.ResponsesC():
				received++
			default:
				assert.Equal(t, goroutines, received)
				return
			}
		}
	})
}

func TestBridge_RealDispatcher(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		disp, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
			[]pubsub.LaneConfig{
				lane.NewBlockingLane(pubsub.CentralBoundLane),
			},
		))
		require.NoError(t, err)
		defer disp.Stop()

		b, err := NewBridge(disp)
		require.NoError(t, err)

		msg := message.New(&central.MsgFromSensor{})
		require.NoError(t, disp.Publish(&CentralBoundEvent{Msg: msg}))
		synctest.Wait()

		select {
		case received := <-b.ResponsesC():
			assert.Same(t, msg, received)
		default:
			t.Fatal("expected message on ResponsesC via real dispatcher")
		}
	})
}

func TestBridge_Name(t *testing.T) {
	capturing := &capturingDispatcher{}
	b, err := NewBridge(capturing)
	require.NoError(t, err)
	assert.Equal(t, "centralbound.Bridge", b.Name())
}
