package centralproxy

import (
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestOnlineOfflineDispatcher(tb testing.TB) common.PubSubDispatcher {
	tb.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
		lane.NewBlockingLane(pubsub.SensorOnlineLane),
		lane.NewBlockingLane(pubsub.SensorOfflineLane),
	}))
	require.NoError(tb, err)
	tb.Cleanup(dispatcher.Stop)
	return dispatcher
}

func TestHandleSensorOnlineEvent_SetsCentralReachable(t *testing.T) {
	h := &Handler{pubSubDispatcher: newTestOnlineOfflineDispatcher(t)}
	require.False(t, h.centralReachable.Load())

	require.NoError(t, h.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.True(t, h.centralReachable.Load())
}

func TestHandleSensorOfflineEvent_UnsetsCentralReachable(t *testing.T) {
	h := &Handler{pubSubDispatcher: newTestOnlineOfflineDispatcher(t)}
	h.centralReachable.Store(true)

	require.NoError(t, h.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.False(t, h.centralReachable.Load())
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	h := &Handler{pubSubDispatcher: newTestOnlineOfflineDispatcher(t)}

	err := h.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	h := &Handler{pubSubDispatcher: newTestOnlineOfflineDispatcher(t)}

	err := h.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestNewProxyHandler_RegistersSensorOnlineOfflineConsumers verifies
// NewProxyHandler() wires the online/offline handlers onto the
// SensorOnline/SensorOffline topic and lane end-to-end through a real
// dispatcher, at construction time (there is no Start() for a
// Notifiable-only component).
func TestNewProxyHandler_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		h := &Handler{pubSubDispatcher: dispatcher}
		require.NoError(t, dispatcher.RegisterConsumerToLane(
			pubsub.CentralProxyHandlerSensorOnlineConsumer,
			pubsub.SensorOnlineTopic,
			pubsub.SensorOnlineLane,
			h.handleSensorOnlineEvent,
		))
		require.NoError(t, dispatcher.RegisterConsumerToLane(
			pubsub.CentralProxyHandlerSensorOfflineConsumer,
			pubsub.SensorOfflineTopic,
			pubsub.SensorOfflineLane,
			h.handleSensorOfflineEvent,
		))

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()
		assert.True(t, h.centralReachable.Load())

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
		assert.False(t, h.centralReachable.Load())
	})
}
