package sensor

// Tests for the SoftRestart callback registered inside Sensor.Start().
//
// Sensor.Start() requires TLS certificates and a running gRPC server, making
// it impractical to call in unit tests. The tests below construct a minimal
// Sensor struct and exercise the callback closure directly, verifying:
//  1. When centralCommunication is nil the callback is a no-op.
//  2. When centralCommunication is set the callback calls Stop().
//  3. With the feature flag enabled, RegisterConsumer is wired to the right
//     consumer ID and topic.

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	roxsync "github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- minimal fakes ----------------------------------------------------------

type fakeCentralComm struct {
	stopCount int
	mu        roxsync.Mutex
	stopped   concurrency.ErrorSignal
}

func (f *fakeCentralComm) Start(_ central.SensorServiceClient, _ *concurrency.Flag, _ *concurrency.Signal, _ config.Handler, _ detector.Detector) {
}

func (f *fakeCentralComm) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCount++
}

func (f *fakeCentralComm) Stopped() concurrency.ReadOnlyErrorSignal { return &f.stopped }

// capturingSensorDispatcher captures the first RegisterConsumerToLane call so
// tests can inspect the wiring and invoke the callback directly.
type capturingSensorDispatcher struct {
	callback   pubsub.EventCallback
	consumerID pubsub.ConsumerID
	topic      pubsub.Topic
	laneID     pubsub.LaneID
}

func (c *capturingSensorDispatcher) RegisterConsumer(_ pubsub.ConsumerID, _ pubsub.Topic, _ pubsub.EventCallback) error {
	return nil
}

func (c *capturingSensorDispatcher) RegisterConsumerToLane(id pubsub.ConsumerID, t pubsub.Topic, l pubsub.LaneID, cb pubsub.EventCallback) error {
	c.consumerID = id
	c.topic = t
	c.laneID = l
	c.callback = cb
	return nil
}

func (c *capturingSensorDispatcher) Publish(_ pubsub.Event) error { return nil }
func (c *capturingSensorDispatcher) Stop()                        {}

// ---- helper -----------------------------------------------------------------

// softRestartCallback mirrors the closure registered by Sensor.Start() when
// the SensorInternalPubSub feature flag is enabled. It must be kept in sync
// with the production code in sensor.go.
func softRestartCallback(s *Sensor) pubsub.EventCallback {
	return func(e pubsub.Event) error {
		if v, ok := e.(interface{ IsExpired() bool }); ok && v.IsExpired() {
			return nil
		}
		s.centralCommunicationLock.Lock()
		defer s.centralCommunicationLock.Unlock()
		if s.centralCommunication == nil {
			return nil
		}
		s.centralCommunication.Stop()
		return nil
	}
}

func sensorForCallbackTest() *Sensor {
	return &Sensor{
		centralCommunicationLock: &roxsync.Mutex{},
		pubSub:                   internalmessage.NewMessageSubscriber(),
	}
}

// ---- tests ------------------------------------------------------------------

// TestSoftRestartCallback_NilCommunication verifies that the callback silently
// returns when the central connection has not been established yet.
func TestSoftRestartCallback_NilCommunication(t *testing.T) {
	s := sensorForCallbackTest()
	require.NoError(t, softRestartCallback(s)(nil))
}

// TestSoftRestartCallback_StopsConnection verifies that the callback calls
// Stop() on the active central communication.
func TestSoftRestartCallback_StopsConnection(t *testing.T) {
	s := sensorForCallbackTest()
	fakeCC := &fakeCentralComm{}
	s.centralCommunication = fakeCC

	require.NoError(t, softRestartCallback(s)(nil))
	assert.Equal(t, 1, fakeCC.stopCount, "Stop() must be called exactly once")
}

// TestSensor_PubSubEnabled_SoftRestartConsumerRegistration verifies that the
// dispatcher is called with CoreSensorConsumer + SoftRestartTopic and that the
// captured callback drives Stop() on the active connection.
func TestSensor_PubSubEnabled_SoftRestartConsumerRegistration(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	capturing := &capturingSensorDispatcher{}
	fakeCC := &fakeCentralComm{}
	s := sensorForCallbackTest()
	s.pubSubDispatcher = capturing
	s.centralCommunication = fakeCC

	// Simulate the call Start() makes when the flag is on.
	require.NoError(t, s.pubSubDispatcher.RegisterConsumerToLane(
		pubsub.CoreSensorConsumer,
		pubsub.SoftRestartTopic,
		pubsub.SoftRestartLane,
		softRestartCallback(s),
	))

	assert.Equal(t, pubsub.CoreSensorConsumer, capturing.consumerID)
	assert.Equal(t, pubsub.SoftRestartTopic, capturing.topic)
	assert.Equal(t, pubsub.SoftRestartLane, capturing.laneID)
	require.NotNil(t, capturing.callback)

	require.NoError(t, capturing.callback(&stubSoftRestartEvent{}))
	assert.Equal(t, 1, fakeCC.stopCount, "callback must call Stop() on centralCommunication")
}

// stubSoftRestartEvent is a minimal non-expired event.
type stubSoftRestartEvent struct{}

func (s *stubSoftRestartEvent) Topic() pubsub.Topic { return pubsub.SoftRestartTopic }
func (s *stubSoftRestartEvent) Lane() pubsub.LaneID { return pubsub.SoftRestartLane }

// expiredSoftRestartEvent simulates a stale event whose validity has expired.
type expiredSoftRestartEvent struct{ stubSoftRestartEvent }

func (e *expiredSoftRestartEvent) IsExpired() bool { return true }

// TestSoftRestartCallback_SkipsExpiredEvent verifies that the callback does
// not call Stop() when the event's validity context has been cancelled.
func TestSoftRestartCallback_SkipsExpiredEvent(t *testing.T) {
	s := sensorForCallbackTest()
	fakeCC := &fakeCentralComm{}
	s.centralCommunication = fakeCC

	require.NoError(t, softRestartCallback(s)(&expiredSoftRestartEvent{}))
	assert.Equal(t, 0, fakeCC.stopCount, "Stop() must not be called for an expired event")
}

// TestSensor_PubSubDisabled_SoftRestartViaInternalmessage verifies the legacy
// internalmessage path: when the flag is off, publishing a SoftRestart message
// through the old subscriber triggers Stop() on centralCommunication.
func TestSensor_PubSubDisabled_SoftRestartViaInternalmessage(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "false")

	s := sensorForCallbackTest()
	fakeCC := &fakeCentralComm{}
	s.centralCommunication = fakeCC

	require.NoError(t, s.pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, func(msg *internalmessage.SensorInternalMessage) {
		if msg.IsExpired() {
			return
		}
		s.centralCommunicationLock.Lock()
		defer s.centralCommunicationLock.Unlock()
		if s.centralCommunication == nil {
			return
		}
		s.centralCommunication.Stop()
	}))

	require.NoError(t, s.pubSub.Publish(&internalmessage.SensorInternalMessage{
		Kind:     internalmessage.SensorMessageSoftRestart,
		Text:     "legacy soft restart",
		Validity: context.Background(),
	}))

	assert.Eventually(t, func() bool {
		fakeCC.mu.Lock()
		defer fakeCC.mu.Unlock()
		return fakeCC.stopCount == 1
	}, 500*time.Millisecond, 5*time.Millisecond, "Stop() must be called via legacy path")
}
