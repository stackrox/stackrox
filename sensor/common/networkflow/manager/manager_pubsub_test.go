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

// capturingDispatcher records the RegisterConsumer call so tests can inspect
// and invoke the registered callback.
type capturingDispatcher struct {
	consumerID  pubsub.ConsumerID
	topic       pubsub.Topic
	callback    pubsub.EventCallback
	registerErr error
}

func (c *capturingDispatcher) RegisterConsumer(id pubsub.ConsumerID, t pubsub.Topic, cb pubsub.EventCallback) error {
	c.consumerID = id
	c.topic = t
	c.callback = cb
	return c.registerErr
}

func (c *capturingDispatcher) RegisterConsumerToLane(_ pubsub.ConsumerID, _ pubsub.Topic, _ pubsub.LaneID, _ pubsub.EventCallback) error {
	return nil
}

func (c *capturingDispatcher) Publish(_ pubsub.Event) error { return nil }
func (c *capturingDispatcher) Stop()                        {}

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

	require.NotNil(t, capturing.callback, "expected RegisterConsumer to be called with a non-nil callback")
	assert.Equal(t, pubsub.NetworkFlowManagerConsumer, capturing.consumerID)
	assert.Equal(t, pubsub.ResourceSyncFinishedTopic, capturing.topic)

	assert.False(t, mgr.initialSync.Load(), "initialSync must be false before the event fires")
	require.NoError(t, capturing.callback(nil))
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
	require.NoError(t, capturing.callback(nil))
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
