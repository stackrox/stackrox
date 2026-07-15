package clustermetrics

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
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

func newTestClusterMetricsWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher, pollInterval time.Duration) *clusterMetricsImpl {
	tb.Helper()
	cm := NewWithInterval(&fakeClusterIDPeeker{}, fake.NewClientset(), pollInterval, dispatcher)
	return cm.(*clusterMetricsImpl)
}

func TestHandleSensorOnlineEvent_ResetsPollTicker(t *testing.T) {
	cm := newTestClusterMetricsWithDispatcher(t, newTestOnlineOfflineDispatcher(t), time.Millisecond*10)

	require.NoError(t, cm.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	select {
	case <-cm.pollTicker.C:
		// Ticker fired: it was reset (running) as expected.
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected pollTicker to be reset and fire")
	}
}

func TestHandleSensorOfflineEvent_StopsPollTicker(t *testing.T) {
	cm := newTestClusterMetricsWithDispatcher(t, newTestOnlineOfflineDispatcher(t), time.Millisecond*10)
	cm.pollTicker.Reset(cm.pollingInterval) // Simulate a previously-running ticker.

	require.NoError(t, cm.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	select {
	case <-cm.pollTicker.C:
		t.Fatal("expected pollTicker to be stopped, but it fired")
	case <-time.After(50 * time.Millisecond):
		// No fire: ticker was stopped as expected.
	}
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	cm := newTestClusterMetricsWithDispatcher(t, newTestOnlineOfflineDispatcher(t), time.Millisecond*10)

	err := cm.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	cm := newTestClusterMetricsWithDispatcher(t, newTestOnlineOfflineDispatcher(t), time.Millisecond*10)

	err := cm.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestClusterMetricsStart_RegistersSensorOnlineOfflineConsumers verifies
// Start() wires the online/offline handlers onto the SensorOnline/
// SensorOffline topic and lane end-to-end through a real dispatcher.
func TestClusterMetricsStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		cm := newTestClusterMetricsWithDispatcher(t, dispatcher, time.Millisecond*10)
		require.NoError(t, cm.Start())
		defer cm.Stop()

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()

		select {
		case resp := <-cm.ResponsesC():
			assert.NotNil(t, resp)
		case <-time.After(time.Second):
			t.Fatal("expected cluster metrics to emit a response after SensorOnline ticker reset")
		}

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
	})
}
