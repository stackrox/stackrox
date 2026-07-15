package scannerclient

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeScannerClient struct {
	closed bool
}

func (f *fakeScannerClient) GetImageAnalysis(context.Context, *storage.Image, *types.Config) (*ImageAnalysis, error) {
	return nil, nil
}

func (f *fakeScannerClient) Close() error {
	f.closed = true
	return nil
}

func newTestOnlineDispatcher(tb testing.TB) common.PubSubDispatcher {
	tb.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
		lane.NewBlockingLane(pubsub.SensorOnlineLane),
	}))
	require.NoError(tb, err)
	tb.Cleanup(dispatcher.Stop)
	return dispatcher
}

// resetSingleton clears the package-level ResetNotifiable singleton so tests
// don't leak state (and don't skip registration) across each other.
func resetSingleton(tb testing.TB) {
	tb.Helper()
	notifiable = nil
	notifiableOnce = sync.Once{}
}

func TestHandleSensorOnlineEvent_ResetsScannerClient(t *testing.T) {
	fake := &fakeScannerClient{}
	scannerClient = fake
	defer func() { scannerClient = nil }()

	r := &resetNotifiable{pubSubDispatcher: newTestOnlineDispatcher(t)}

	require.NoError(t, r.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.True(t, fake.closed)
	assert.Nil(t, scannerClient)
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	r := &resetNotifiable{pubSubDispatcher: newTestOnlineDispatcher(t)}

	err := r.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestResetNotifiable_RegistersSensorOnlineConsumer verifies
// ResetNotifiable() wires the online handler onto the SensorOnline topic and
// lane end-to-end through a real dispatcher, at construction time (there is
// no Start() for a Notifiable-only component).
func TestResetNotifiable_RegistersSensorOnlineConsumer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resetSingleton(t)
		defer resetSingleton(t)

		fake := &fakeScannerClient{}
		scannerClient = fake
		defer func() { scannerClient = nil }()

		dispatcher := newTestOnlineDispatcher(t)
		ResetNotifiable(dispatcher)

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()

		assert.True(t, fake.closed)
		assert.Nil(t, scannerClient)
	})
}
