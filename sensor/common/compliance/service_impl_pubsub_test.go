package compliance

import (
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServiceWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher) *serviceImpl {
	tb.Helper()
	offlineMode := &atomic.Bool{}
	offlineMode.Store(true)
	return &serviceImpl{
		connectionManager: newConnectionManager(),
		offlineMode:       offlineMode,
		stopper:           set.NewSet[concurrency.Stopper](),
		pubSubDispatcher:  dispatcher,
	}
}

func TestServiceHandleSensorOnlineEvent_ClearsOfflineMode(t *testing.T) {
	s := newTestServiceWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	require.True(t, s.offlineMode.Load())

	require.NoError(t, s.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.False(t, s.offlineMode.Load())
}

func TestServiceHandleSensorOfflineEvent_SetsOfflineMode(t *testing.T) {
	s := newTestServiceWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	s.offlineMode.Store(false)

	require.NoError(t, s.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.True(t, s.offlineMode.Load())
}

func TestServiceHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	s := newTestServiceWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := s.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestServiceHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	s := newTestServiceWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := s.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestServiceStart_RegistersSensorOnlineOfflineConsumers verifies Start()
// wires the online/offline handlers onto the SensorOnline/SensorOffline
// topic and lane end-to-end through a real dispatcher.
func TestServiceStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		s := newTestServiceWithDispatcher(t, dispatcher)
		require.NoError(t, s.Start())
		defer s.Stop()

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()
		assert.False(t, s.offlineMode.Load())

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
		assert.True(t, s.offlineMode.Load())
	})
}
