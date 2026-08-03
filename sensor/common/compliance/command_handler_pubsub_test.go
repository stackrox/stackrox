package compliance

import (
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/mocks"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
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

func newTestCommandHandlerWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher) *commandHandlerImpl {
	tb.Helper()
	mockService := mocks.NewMockService(gomock.NewController(tb))
	mockService.EXPECT().Output().AnyTimes().DoAndReturn(func() chan *compliance.ComplianceReturn {
		return make(chan *compliance.ComplianceReturn)
	})
	return &commandHandlerImpl{
		service:          mockService,
		commands:         make(chan *central.ScrapeCommand),
		updates:          make(chan *message.ExpiringMessage),
		scrapeIDToState:  make(map[string]*scrapeState),
		stopper:          concurrency.NewStopper(),
		pubSubDispatcher: dispatcher,
	}
}

func TestCommandHandlerHandleSensorOnlineEvent_SetsCentralReachable(t *testing.T) {
	c := newTestCommandHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	require.False(t, c.centralReachable.Load())

	require.NoError(t, c.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.True(t, c.centralReachable.Load())
}

func TestCommandHandlerHandleSensorOfflineEvent_UnsetsCentralReachable(t *testing.T) {
	c := newTestCommandHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	c.centralReachable.Store(true)

	require.NoError(t, c.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.False(t, c.centralReachable.Load())
}

func TestCommandHandlerHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	c := newTestCommandHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := c.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestCommandHandlerHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	c := newTestCommandHandlerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := c.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestCommandHandlerStart_RegistersSensorOnlineOfflineConsumers verifies
// Start() wires the online/offline handlers onto the SensorOnline/
// SensorOffline topic and lane end-to-end through a real dispatcher.
func TestCommandHandlerStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		c := newTestCommandHandlerWithDispatcher(t, dispatcher)
		require.NoError(t, c.Start())
		defer c.Stop()

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()
		assert.True(t, c.centralReachable.Load())

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
		assert.False(t, c.centralReachable.Load())
	})
}
