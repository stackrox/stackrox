package compliance

import (
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/index"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestNodeInventoryHandlerWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher) *nodeInventoryHandlerImpl {
	tb.Helper()
	inventories := make(chan *storage.NodeInventory)
	reports := make(chan *index.IndexReportWrap)
	tb.Cleanup(func() {
		close(inventories)
		close(reports)
	})
	return NewNodeInventoryHandler(inventories, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{}, dispatcher)
}

func TestNodeInventoryHandlerHandleSensorOnlineEvent_SignalsCentralReady(t *testing.T) {
	h := newTestNodeInventoryHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	require.False(t, h.centralReady.IsDone())

	require.NoError(t, h.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.True(t, h.centralReady.IsDone())
}

func TestNodeInventoryHandlerHandleSensorOfflineEvent_ResetsCentralReady(t *testing.T) {
	h := newTestNodeInventoryHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	h.centralReady.Signal()
	require.True(t, h.centralReady.IsDone())

	require.NoError(t, h.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.False(t, h.centralReady.IsDone())
}

func TestNodeInventoryHandlerHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	h := newTestNodeInventoryHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := h.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestNodeInventoryHandlerHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	h := newTestNodeInventoryHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := h.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestNodeInventoryHandlerStart_RegistersSensorOnlineOfflineConsumers
// verifies Start() wires the online/offline handlers onto the
// SensorOnline/SensorOffline topic and lane end-to-end through a real
// dispatcher.
func TestNodeInventoryHandlerStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		h := newTestNodeInventoryHandlerWithDispatcher(t, dispatcher)
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
