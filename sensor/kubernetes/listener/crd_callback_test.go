package listener

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/pubsub"
	listenerMocks "github.com/stackrox/rox/sensor/kubernetes/listener/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestCrdWatcherCallbackWrapper_PubSubEnabled_PublishesSoftRestartEvent verifies
// that when the feature flag is on and the condition is met, the callback
// publishes a SoftRestartEvent via the pubsub dispatcher.
func TestCrdWatcherCallbackWrapper_PubSubEnabled_PublishesSoftRestartEvent(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	const expectedText = "test soft restart"
	mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)

	var capturedEvent pubsub.Event
	mockDispatcher.EXPECT().Publish(gomock.Any()).DoAndReturn(func(e pubsub.Event) error {
		capturedEvent = e
		return nil
	})

	cb := crdWatcherCallbackWrapper(
		context.Background(),
		allResourcesAvailable(),
		internalmessage.NewMessageSubscriber(),
		mockDispatcher,
		expectedText,
	)
	cb(&watcher.Status{Available: true})

	require.IsType(t, &SoftRestartEvent{}, capturedEvent)
	evt := capturedEvent.(*SoftRestartEvent)
	assert.Equal(t, expectedText, evt.Text)
}

// TestCrdWatcherCallbackWrapper_PubSubEnabled_ConditionNotMet_DoesNotPublish verifies
// that the callback is a no-op when the callback condition is not satisfied.
func TestCrdWatcherCallbackWrapper_PubSubEnabled_ConditionNotMet_DoesNotPublish(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// No EXPECT() on mockDispatcher — any Publish call would fail the test.
	mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)

	cb := crdWatcherCallbackWrapper(
		context.Background(),
		allResourcesAvailable(), // expects Available == true
		internalmessage.NewMessageSubscriber(),
		mockDispatcher,
		"should not fire",
	)
	cb(&watcher.Status{Available: false}) // condition not satisfied
}

// TestCrdWatcherCallbackWrapper_PubSubDisabled_PublishesViaInternalmessage verifies
// that when the feature flag is off, the callback uses the legacy internalmessage
// path instead of pubsub.
func TestCrdWatcherCallbackWrapper_PubSubDisabled_PublishesViaInternalmessage(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "false")

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// No EXPECT() — pubsub dispatcher must NOT be called when flag is off.
	mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)

	pubSub := internalmessage.NewMessageSubscriber()
	const expectedText = "legacy restart path"

	received := make(chan *internalmessage.SensorInternalMessage, 1)
	require.NoError(t, pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, func(msg *internalmessage.SensorInternalMessage) {
		received <- msg
	}))

	cb := crdWatcherCallbackWrapper(
		context.Background(),
		allResourcesAvailable(),
		pubSub,
		mockDispatcher,
		expectedText,
	)
	cb(&watcher.Status{Available: true})

	select {
	case msg := <-received:
		assert.Equal(t, expectedText, msg.Text)
		assert.Equal(t, internalmessage.SensorMessageSoftRestart, msg.Kind)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout: internalmessage SoftRestart callback never fired")
	}
}

// TestCrdWatcherCallbackWrapper_PubSubEnabled_ResourcesUnavailable verifies
// that the callback also fires when using the resourcesUnavailable condition
// (i.e., when a CRD is removed and status reports unavailable).
func TestCrdWatcherCallbackWrapper_PubSubEnabled_ResourcesUnavailable(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	const expectedText = "resources removed"
	mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)

	var capturedEvent pubsub.Event
	mockDispatcher.EXPECT().Publish(gomock.Any()).DoAndReturn(func(e pubsub.Event) error {
		capturedEvent = e
		return nil
	})

	cb := crdWatcherCallbackWrapper(
		context.Background(),
		resourcesUnavailable(),
		internalmessage.NewMessageSubscriber(),
		mockDispatcher,
		expectedText,
	)
	cb(&watcher.Status{Available: false})

	require.IsType(t, &SoftRestartEvent{}, capturedEvent)
	assert.Equal(t, expectedText, capturedEvent.(*SoftRestartEvent).Text)
}

// TestCrdWatcherCallbackWrapper_PubSubEnabled_ResourcesUnavailable_ConditionNotMet verifies
// the unavailable condition does not fire when resources are available.
func TestCrdWatcherCallbackWrapper_PubSubEnabled_ResourcesUnavailable_ConditionNotMet(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)

	cb := crdWatcherCallbackWrapper(
		context.Background(),
		resourcesUnavailable(),
		internalmessage.NewMessageSubscriber(),
		mockDispatcher,
		"should not fire",
	)
	cb(&watcher.Status{Available: true}) // condition NOT met for resourcesUnavailable
}

// TestCrdWatcherCallbackWrapper_PubSubEnabled_CancelledContext verifies that
// when the context is cancelled before the CRD status fires, the published
// SoftRestartEvent carries the cancelled context and reports IsExpired() == true.
func TestCrdWatcherCallbackWrapper_PubSubEnabled_CancelledContext(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var capturedEvent pubsub.Event
	mockDispatcher.EXPECT().Publish(gomock.Any()).DoAndReturn(func(e pubsub.Event) error {
		capturedEvent = e
		return nil
	})

	cb := crdWatcherCallbackWrapper(
		ctx,
		allResourcesAvailable(),
		internalmessage.NewMessageSubscriber(),
		mockDispatcher,
		"cancelled restart",
	)
	cb(&watcher.Status{Available: true})

	require.IsType(t, &SoftRestartEvent{}, capturedEvent)
	assert.True(t, capturedEvent.(*SoftRestartEvent).IsExpired(), "event must be expired when context is cancelled")
}
