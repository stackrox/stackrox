package admissioncontroller

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

func TestHandleSensorOnlineEvent_SignalsCentralReady(t *testing.T) {
	h := newAlertHandler(newTestOnlineOfflineDispatcher(t))
	require.False(t, h.centralReady.IsDone())

	require.NoError(t, h.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.True(t, h.centralReady.IsDone())
}

func TestHandleSensorOfflineEvent_ResetsCentralReady(t *testing.T) {
	h := newAlertHandler(newTestOnlineOfflineDispatcher(t))
	h.centralReady.Signal()
	require.True(t, h.centralReady.IsDone())

	require.NoError(t, h.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.False(t, h.centralReady.IsDone())
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	h := newAlertHandler(newTestOnlineOfflineDispatcher(t))

	err := h.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	h := newAlertHandler(newTestOnlineOfflineDispatcher(t))

	err := h.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestAlertHandlerStart_RegistersSensorOnlineOfflineConsumers verifies
// Start() wires the online/offline handlers onto the SensorOnline/
// SensorOffline topic and lane end-to-end through a real dispatcher, and
// that legacy Notify no longer double-drives centralReady once PubSub is
// enabled.
func TestAlertHandlerStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		h := newAlertHandler(dispatcher)
		require.NoError(t, h.Start())
		defer h.Stop()

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()
		assert.True(t, h.centralReady.IsDone())

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
		assert.False(t, h.centralReady.IsDone())

		// Legacy Notify must be a no-op for these events in PubSub mode.
		h.Notify(common.SensorComponentEventCentralReachable)
		assert.False(t, h.centralReady.IsDone())
	})
}
