package index

import (
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stackrox/rox/sensor/common/virtualmachine/index/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func newTestHandlerWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher) *handlerImpl {
	tb.Helper()
	ctrl := gomock.NewController(tb)
	store := mocks.NewMockVirtualMachineStore(ctrl)
	return &handlerImpl{
		centralReady:     concurrency.NewSignal(),
		lock:             &sync.RWMutex{},
		stopper:          concurrency.NewStopper(),
		store:            store,
		pubSubDispatcher: dispatcher,
	}
}

func TestHandleSensorOnlineEvent_SignalsCentralReady(t *testing.T) {
	h := newTestHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	require.False(t, h.centralReady.IsDone())

	require.NoError(t, h.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.True(t, h.centralReady.IsDone())
}

func TestHandleSensorOfflineEvent_ResetsCentralReady(t *testing.T) {
	h := newTestHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	h.centralReady.Signal()
	require.True(t, h.centralReady.IsDone())

	require.NoError(t, h.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.False(t, h.centralReady.IsDone())
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	h := newTestHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := h.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	h := newTestHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := h.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestHandlerStart_RegistersSensorOnlineOfflineConsumers verifies Start()
// wires the online/offline handlers onto the SensorOnline/SensorOffline
// topic and lane end-to-end through a real dispatcher.
func TestHandlerStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		h := newTestHandlerWithDispatcher(t, dispatcher)
		require.NoError(t, h.Start())
		defer h.Stop()

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()
		assert.True(t, h.centralReady.IsDone())

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
		assert.False(t, h.centralReady.IsDone())
	})
}
