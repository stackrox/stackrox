package manager

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/features"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	mocksExternalSrc "github.com/stackrox/rox/sensor/common/externalsrcs/mocks"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	mocksManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	"github.com/stackrox/rox/sensor/common/networkflow/updatecomputer"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// capturingDispatcher records the RegisterConsumerToLane call so tests can
// inspect and invoke the registered callback.
type capturingDispatcher struct {
	consumerID  pubsub.ConsumerID
	topic       pubsub.Topic
	laneID      pubsub.LaneID
	callback    pubsub.EventCallback
	registerErr error
}

func (c *capturingDispatcher) RegisterConsumer(_ pubsub.ConsumerID, _ pubsub.Topic, _ pubsub.EventCallback) error {
	return nil
}

func (c *capturingDispatcher) RegisterConsumerToLane(id pubsub.ConsumerID, t pubsub.Topic, l pubsub.LaneID, cb pubsub.EventCallback) error {
	c.consumerID = id
	c.topic = t
	c.laneID = l
	c.callback = cb
	return c.registerErr
}

func (c *capturingDispatcher) Publish(_ pubsub.Event) error { return nil }
func (c *capturingDispatcher) Stop()                        {}

// stubEvent is a minimal pubsub.Event that does not implement IsExpired.
type stubEvent struct{}

func (s *stubEvent) Topic() pubsub.Topic { return pubsub.ResourceSyncFinishedTopic }
func (s *stubEvent) Lane() pubsub.LaneID { return pubsub.ResourceSyncFinishedLane }

// expiredEvent implements the IsExpired interface to simulate a stale event.
type expiredEvent struct{ stubEvent }

func (e *expiredEvent) IsExpired() bool { return true }

func newManagerForPubSubTest(t *testing.T, dispatcher *capturingDispatcher) (Manager, *networkFlowManager) {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	mockEntityStore := mocksManager.NewMockEntityStore(mockCtrl)
	mockExternalStore := mocksExternalSrc.NewMockStore(mockCtrl)
	mockDetector := mocksDetector.NewMockDetector(mockCtrl)

	m := NewManager(
		mockEntityStore,
		mockExternalStore,
		mockDetector,
		internalmessage.NewMessageSubscriber(),
		dispatcher,
		updatecomputer.New(),
	)
	return m, m.(*networkFlowManager)
}

// TestNewManager_PubSubEnabled_RegistersResourceSyncConsumer verifies that when
// the pubsub feature flag is enabled, NewManager registers a consumer for
// ResourceSyncFinishedTopic and that firing the callback marks the initial sync.
func TestNewManager_PubSubEnabled_RegistersResourceSyncConsumer(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	capturing := &capturingDispatcher{}
	_, mgr := newManagerForPubSubTest(t, capturing)

	require.NotNil(t, capturing.callback, "expected RegisterConsumerToLane to be called with a non-nil callback")
	assert.Equal(t, pubsub.NetworkFlowManagerConsumer, capturing.consumerID)
	assert.Equal(t, pubsub.ResourceSyncFinishedTopic, capturing.topic)
	assert.Equal(t, pubsub.ResourceSyncFinishedLane, capturing.laneID)

	assert.False(t, mgr.initialSync.Load(), "initialSync must be false before the event fires")
	require.NoError(t, capturing.callback(&stubEvent{}))
	assert.True(t, mgr.initialSync.Load(), "initialSync must be true after ResourceSyncFinished fires")
}

// TestNewManager_PubSubEnabled_CallbackHonorsStopper verifies the stop-guard: when
// the manager's stopper has been requested, the callback exits early without
// marking the initial sync.
func TestNewManager_PubSubEnabled_CallbackHonorsStopper(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	capturing := &capturingDispatcher{}
	_, mgr := newManagerForPubSubTest(t, capturing)

	require.NotNil(t, capturing.callback)
	mgr.stopper.Client().Stop()
	require.NoError(t, capturing.callback(&stubEvent{}))
	assert.False(t, mgr.initialSync.Load(), "callback must not set initialSync when stopper is triggered")
}

// TestNewManager_PubSubDisabled_SubscribesViaInternalmessage verifies that when
// the pubsub feature flag is off, NewManager uses the old internalmessage
// subscription path, and triggering it still marks the initial sync.
func TestNewManager_PubSubDisabled_SubscribesViaInternalmessage(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "false")
	capturing := &capturingDispatcher{}
	_, mgr := newManagerForPubSubTest(t, capturing)

	assert.Nil(t, capturing.callback, "RegisterConsumer must NOT be called when pubsub flag is off")
	assert.False(t, mgr.initialSync.Load())

	require.NoError(t, mgr.pubSub.Publish(&internalmessage.SensorInternalMessage{
		Kind:     internalmessage.SensorMessageResourceSyncFinished,
		Text:     "test sync",
		Validity: context.Background(),
	}))

	// Publish dispatches goroutines; wait up to 500ms for the callback to fire.
	assert.Eventually(t, func() bool {
		return mgr.initialSync.Load()
	}, 500*time.Millisecond, 5*time.Millisecond, "initialSync must be set after internalmessage publish")
}

// TestNewManager_PubSubEnabled_CallbackSkipsExpiredEvent verifies that the
// callback drops stale events whose validity context has been cancelled.
func TestNewManager_PubSubEnabled_CallbackSkipsExpiredEvent(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	capturing := &capturingDispatcher{}
	_, mgr := newManagerForPubSubTest(t, capturing)

	require.NotNil(t, capturing.callback)
	require.NoError(t, capturing.callback(&expiredEvent{}))
	assert.False(t, mgr.initialSync.Load(), "callback must not set initialSync for an expired event")
}
