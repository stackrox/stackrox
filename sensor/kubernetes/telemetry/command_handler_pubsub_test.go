package telemetry

import (
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
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

func newTestCommandHandlerWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher) *commandHandler {
	tb.Helper()
	return newCommandHandler(fake.NewClientset(), resources.InitializeStore(nil), dispatcher)
}

func TestHandleSensorOnlineEvent_SetsCentralReachable(t *testing.T) {
	h := newTestCommandHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	require.False(t, h.centralReachable.Load())

	require.NoError(t, h.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.True(t, h.centralReachable.Load())
}

func TestHandleSensorOfflineEvent_UnsetsCentralReachable(t *testing.T) {
	h := newTestCommandHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	h.centralReachable.Store(true)

	require.NoError(t, h.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.False(t, h.centralReachable.Load())
}

func TestHandleSensorOfflineEvent_ClearsPendingRequests(t *testing.T) {
	h := newTestCommandHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	h.pendingContextCancels["req-1"] = func() {}

	require.NoError(t, h.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	// cancelPendingRequests() only clears bookkeeping without invoking the
	// stored cancel funcs (pre-existing behavior of the legacy Notify path,
	// unchanged by this migration); just assert it runs the same way here.
	assert.Empty(t, h.pendingContextCancels)
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	h := newTestCommandHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := h.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	h := newTestCommandHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := h.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestCommandHandlerStart_RegistersSensorOnlineOfflineConsumers verifies
// Start() wires the online/offline handlers onto the SensorOnline/
// SensorOffline topic and lane end-to-end through a real dispatcher.
func TestCommandHandlerStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		h := newTestCommandHandlerWithDispatcher(t, dispatcher)
		require.NoError(t, h.Start())
		defer h.Stop()

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()
		assert.True(t, h.centralReachable.Load())

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
		assert.False(t, h.centralReachable.Load())
	})
}
