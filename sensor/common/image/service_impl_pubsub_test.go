package image

import (
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/pkg/concurrency"
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

func newTestServiceWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher) *serviceImpl {
	tb.Helper()
	return &serviceImpl{
		centralReady:     concurrency.NewSignal(),
		pubSubDispatcher: dispatcher,
	}
}

func TestHandleSensorOnlineEvent_SignalsCentralReady(t *testing.T) {
	s := newTestServiceWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	require.False(t, s.centralReady.IsDone())

	require.NoError(t, s.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.True(t, s.centralReady.IsDone())
}

func TestHandleSensorOfflineEvent_ResetsCentralReady(t *testing.T) {
	s := newTestServiceWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	s.centralReady.Signal()
	require.True(t, s.centralReady.IsDone())

	require.NoError(t, s.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.False(t, s.centralReady.IsDone())
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	s := newTestServiceWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := s.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
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
		assert.True(t, s.centralReady.IsDone())

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
		assert.False(t, s.centralReady.IsDone())
	})
}
