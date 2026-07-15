package clusterhealth

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

func newTestUpdaterWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher, updateInterval time.Duration) *updaterImpl {
	tb.Helper()
	u := NewUpdater(fake.NewClientset(), updateInterval, dispatcher)
	return u.(*updaterImpl)
}

func TestHandleSensorOnlineEvent_ResetsTicker(t *testing.T) {
	u := newTestUpdaterWithDispatcher(t, newTestOnlineOfflineDispatcher(t), time.Millisecond*10)

	require.NoError(t, u.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	select {
	case <-u.updateTicker.C:
		// Ticker fired: it was reset (running) as expected.
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected ticker to be reset and fire")
	}
}

func TestHandleSensorOfflineEvent_StopsTicker(t *testing.T) {
	u := newTestUpdaterWithDispatcher(t, newTestOnlineOfflineDispatcher(t), time.Millisecond*10)
	u.updateTicker.Reset(u.updateInterval) // Simulate a previously-running ticker.

	require.NoError(t, u.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	select {
	case <-u.updateTicker.C:
		t.Fatal("expected ticker to be stopped, but it fired")
	case <-time.After(50 * time.Millisecond):
		// No fire: ticker was stopped as expected.
	}
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	u := newTestUpdaterWithDispatcher(t, newTestOnlineOfflineDispatcher(t), time.Millisecond*10)

	err := u.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	u := newTestUpdaterWithDispatcher(t, newTestOnlineOfflineDispatcher(t), time.Millisecond*10)

	err := u.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestUpdaterStart_RegistersSensorOnlineOfflineConsumers verifies Start()
// wires the online/offline handlers onto the SensorOnline/SensorOffline
// topic and lane end-to-end through a real dispatcher.
func TestUpdaterStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		u := newTestUpdaterWithDispatcher(t, dispatcher, time.Millisecond*10)
		require.NoError(t, u.Start())
		defer u.Stop()

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()

		select {
		case resp := <-u.ResponsesC():
			assert.NotNil(t, resp)
		case <-time.After(time.Second):
			t.Fatal("expected updater to emit a response after SensorOnline ticker reset")
		}

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
	})
}
