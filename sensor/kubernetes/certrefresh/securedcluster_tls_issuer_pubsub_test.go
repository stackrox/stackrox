package certrefresh

import (
	"testing"

	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
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

func TestHandleSensorOnlineEvent_ActivatesWhenCapabilityPresent(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.SecuredClusterCertificatesReissue})
	defer centralcaps.Set([]centralsensor.CentralCapability{})

	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.tlsIssuer.pubSubDispatcher = newTestOnlineOfflineDispatcher(t)
	fixture.tlsIssuer.started.Store(true)
	fixture.mockForStart(mockForStartConfig{})

	require.NoError(t, fixture.tlsIssuer.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.True(t, fixture.tlsIssuer.online.Load())
	fixture.assertMockExpectations(t)
}

func TestHandleSensorOnlineEvent_SkipsActivationWhenCapabilityMissing(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{})
	defer centralcaps.Set([]centralsensor.CentralCapability{})

	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.tlsIssuer.pubSubDispatcher = newTestOnlineOfflineDispatcher(t)
	fixture.tlsIssuer.started.Store(true)

	require.NoError(t, fixture.tlsIssuer.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	// online is never set, and none of the mocked dependencies (certRefresher,
	// componentGetter) are called, since the capability check short-circuits first.
	assert.False(t, fixture.tlsIssuer.online.Load())
	fixture.assertMockExpectations(t)
}

func TestHandleSensorOfflineEvent_Deactivates(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.SecuredClusterCertificatesReissue})
	defer centralcaps.Set([]centralsensor.CentralCapability{})

	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.tlsIssuer.pubSubDispatcher = newTestOnlineOfflineDispatcher(t)
	fixture.tlsIssuer.started.Store(true)
	fixture.mockForStart(mockForStartConfig{})
	require.NoError(t, fixture.tlsIssuer.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))
	require.True(t, fixture.tlsIssuer.online.Load())

	fixture.certRefresher.On("Stop").Once()
	require.NoError(t, fixture.tlsIssuer.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.False(t, fixture.tlsIssuer.online.Load())
	fixture.assertMockExpectations(t)
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.tlsIssuer.pubSubDispatcher = newTestOnlineOfflineDispatcher(t)

	err := fixture.tlsIssuer.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.tlsIssuer.pubSubDispatcher = newTestOnlineOfflineDispatcher(t)

	err := fixture.tlsIssuer.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestTLSIssuerStart_RegistersSensorOnlineOfflineConsumers verifies Start()
// wires the online/offline handlers onto the SensorOnline/SensorOffline
// topic and lane end-to-end through a real dispatcher.
func TestTLSIssuerStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{})
	defer centralcaps.Set([]centralsensor.CentralCapability{})

	dispatcher := newTestOnlineOfflineDispatcher(t)
	fixture := newSecuredClusterTLSIssuerFixture(fakeK8sClientConfig{})
	fixture.tlsIssuer.pubSubDispatcher = dispatcher

	require.NoError(t, fixture.tlsIssuer.Start())
	defer fixture.tlsIssuer.Stop()

	// No capability, so online() short-circuits before touching any mocks --
	// this only exercises the registration wiring, not activation.
	require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
	require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
}
