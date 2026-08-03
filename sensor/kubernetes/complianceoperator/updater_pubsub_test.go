package complianceoperator

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestSyncFinishedDispatcher(tb testing.TB) common.PubSubDispatcher {
	tb.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{lane.NewBlockingLane(pubsub.SyncFinishedLane)},
	))
	require.NoError(tb, err)
	tb.Cleanup(dispatcher.Stop)
	return dispatcher
}

func newTestUpdaterWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher, updateInterval time.Duration) *updaterImpl {
	tb.Helper()
	readySignal := concurrency.NewSignal()
	u := NewInfoUpdater(fake.NewClientset(), updateInterval, &readySignal, dispatcher)
	return u.(*updaterImpl)
}

func TestHandleSyncFinishedEvent_WrongEventType(t *testing.T) {
	u := newTestUpdaterWithDispatcher(t, newTestSyncFinishedDispatcher(t), time.Millisecond*10)

	err := u.handleSyncFinishedEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSyncFinishedEvent_ResetsTickerWhenCapabilityPresent(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	defer centralcaps.Set([]centralsensor.CentralCapability{})

	u := newTestUpdaterWithDispatcher(t, newTestSyncFinishedDispatcher(t), time.Millisecond*10)

	require.NoError(t, u.handleSyncFinishedEvent(&events.SyncFinishedEvent{}))

	select {
	case <-u.updateTicker.C:
		// Ticker fired: it was reset (running) as expected.
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected ticker to be reset and fire")
	}
}

func TestHandleSyncFinishedEvent_StopsTickerWhenCapabilityAbsent(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{})

	u := newTestUpdaterWithDispatcher(t, newTestSyncFinishedDispatcher(t), time.Millisecond*10)
	u.updateTicker.Reset(u.updateInterval) // Simulate a previously-running ticker.

	require.NoError(t, u.handleSyncFinishedEvent(&events.SyncFinishedEvent{}))

	select {
	case <-u.updateTicker.C:
		t.Fatal("expected ticker to be stopped, but it fired")
	case <-time.After(50 * time.Millisecond):
		// No fire: ticker was stopped as expected.
	}
}

// TestUpdaterStart_RegistersSyncFinishedConsumer verifies Start() wires
// handleSyncFinishedEvent onto the SyncFinished topic/lane end-to-end through
// a real (non-mocked) dispatcher.
func TestUpdaterStart_RegistersSyncFinishedConsumer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
		defer centralcaps.Set([]centralsensor.CentralCapability{})

		dispatcher := newTestSyncFinishedDispatcher(t)
		u := newTestUpdaterWithDispatcher(t, dispatcher, time.Millisecond*10)
		require.NoError(t, u.Start())
		defer u.Stop()

		// Drain the unconditional first response sent by run() at startup.
		<-u.ResponsesC()

		require.NoError(t, dispatcher.Publish(&events.SyncFinishedEvent{}))
		synctest.Wait()

		// The ticker was reset by the PubSub-driven handler; the run loop
		// should eventually emit another response once it fires.
		select {
		case resp := <-u.ResponsesC():
			assert.NotNil(t, resp)
		case <-time.After(time.Second):
			t.Fatal("expected updater to emit a response after SyncFinished ticker reset")
		}
	})
}
