package sensor

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
	require.NoError(t, s.makeSoftRestartCallback()(&pubsub.SoftRestartEvent{Text: "restart"}))
}

// TestSoftRestartCallback_StopsConnection verifies that the callback calls
// Stop() on the active central communication.
func TestSoftRestartCallback_StopsConnection(t *testing.T) {
	s := sensorForCallbackTest()
	fakeCC := &fakeCentralComm{}
	s.centralCommunication = fakeCC

	require.NoError(t, s.makeSoftRestartCallback()(&pubsub.SoftRestartEvent{Text: "restart"}))
	assert.Equal(t, 1, fakeCC.stopCount, "Stop() must be called exactly once")
}

// TestSensor_PubSubEnabled_SoftRestartConsumerRegistration verifies that the
// dispatcher is called with SensorSoftRestartConsumer + SoftRestartTopic and that the
// captured callback drives Stop() on the active connection.
func TestSensor_PubSubEnabled_SoftRestartConsumerRegistration(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	capturing := &capturingSensorDispatcher{}
	fakeCC := &fakeCentralComm{}
	s := sensorForCallbackTest()
	s.pubSubDispatcher = capturing
	s.centralCommunication = fakeCC

	s.registerSoftRestartHandler()

	assert.Equal(t, pubsub.SensorSoftRestartConsumer, capturing.consumerID)
	assert.Equal(t, pubsub.SoftRestartTopic, capturing.topic)
	assert.Equal(t, pubsub.SoftRestartLane, capturing.laneID)
	require.NotNil(t, capturing.callback)

	require.NoError(t, capturing.callback(&pubsub.SoftRestartEvent{Text: "CRD resources changed"}))
	assert.Equal(t, 1, fakeCC.stopCount, "callback must call Stop() on centralCommunication")
}

// TestSoftRestartCallback_SkipsExpiredEvent verifies that the callback does
// not call Stop() when the event's validity context has been cancelled.
func TestSoftRestartCallback_SkipsExpiredEvent(t *testing.T) {
	s := sensorForCallbackTest()
	fakeCC := &fakeCentralComm{}
	s.centralCommunication = fakeCC

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, s.makeSoftRestartCallback()(&pubsub.SoftRestartEvent{
		Text:     "expired restart",
		Validity: ctx,
	}))
	assert.Equal(t, 0, fakeCC.stopCount, "Stop() must not be called for an expired event")
}

// TestSoftRestartCallback_WrongEventType verifies that the callback returns an
// error when it receives an event that is not a *listener.SoftRestartEvent.
func TestSoftRestartCallback_WrongEventType(t *testing.T) {
	s := sensorForCallbackTest()
	fakeCC := &fakeCentralComm{}
	s.centralCommunication = fakeCC

	err := s.makeSoftRestartCallback()(&pubsub.ResourceSyncFinishedEvent{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
	assert.Equal(t, 0, fakeCC.stopCount, "Stop() must not be called for wrong event type")
}

// TestSensor_PubSubDisabled_SoftRestartViaInternalmessage verifies the legacy
// internalmessage path: when the flag is off, publishing a SoftRestart message
// through the old subscriber triggers Stop() on centralCommunication.
func TestSensor_PubSubDisabled_SoftRestartViaInternalmessage(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "false")

	s := sensorForCallbackTest()
	fakeCC := &fakeCentralComm{}
	s.centralCommunication = fakeCC

	require.NoError(t, s.pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, s.makeSoftRestartLegacyHandler()))

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

// TestSoftRestartLegacyHandler_SkipsExpiredMessage verifies that the legacy
// handler ignores messages whose validity context has been cancelled.
func TestSoftRestartLegacyHandler_SkipsExpiredMessage(t *testing.T) {
	s := sensorForCallbackTest()
	fakeCC := &fakeCentralComm{}
	s.centralCommunication = fakeCC

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	handler := s.makeSoftRestartLegacyHandler()
	handler(&internalmessage.SensorInternalMessage{
		Kind:     internalmessage.SensorMessageSoftRestart,
		Text:     "expired restart",
		Validity: ctx,
	})

	assert.Equal(t, 0, fakeCC.stopCount, "Stop() must not be called for an expired message")
}

// TestSoftRestartLegacyHandler_NilCommunication verifies that the legacy
// handler is a no-op when centralCommunication has not been established.
func TestSoftRestartLegacyHandler_NilCommunication(t *testing.T) {
	s := sensorForCallbackTest()

	handler := s.makeSoftRestartLegacyHandler()
	handler(&internalmessage.SensorInternalMessage{
		Kind:     internalmessage.SensorMessageSoftRestart,
		Text:     "no connection yet",
		Validity: context.Background(),
	})
}
