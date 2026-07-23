package listener

import (
	"context"
	"strconv"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/pubsub"
	listenerMocks "github.com/stackrox/rox/sensor/kubernetes/listener/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestCrdWatcherCallback_PublishesSoftRestartEvent verifies that when the
// condition is met, both delivery paths publish a SoftRestart event.
func TestCrdWatcherCallback_PublishesSoftRestartEvent(t *testing.T) {
	tests := map[string]struct {
		pubsubEnabled bool
	}{
		"legacy": {pubsubEnabled: false},
		"pubsub": {pubsubEnabled: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), strconv.FormatBool(tc.pubsubEnabled))

				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				const expectedText = "test soft restart"
				mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)
				pubSub := internalmessage.NewMessageSubscriber()

				if tc.pubsubEnabled {
					var capturedEvent pubsub.Event
					mockDispatcher.EXPECT().Publish(gomock.Any()).DoAndReturn(func(e pubsub.Event) error {
						capturedEvent = e
						return nil
					})

					cb := crdWatcherCallbackWrapper(
						context.Background(),
						allResourcesAvailable(),
						pubSub,
						mockDispatcher,
						expectedText,
					)
					cb(&watcher.Status{Available: true})

					require.IsType(t, &events.SoftRestartEvent{}, capturedEvent)
					evt := capturedEvent.(*events.SoftRestartEvent)
					assert.Equal(t, expectedText, evt.Text)
				} else {
					var received *internalmessage.SensorInternalMessage
					require.NoError(t, pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, func(msg *internalmessage.SensorInternalMessage) {
						received = msg
					}))

					cb := crdWatcherCallbackWrapper(
						context.Background(),
						allResourcesAvailable(),
						pubSub,
						mockDispatcher,
						expectedText,
					)
					cb(&watcher.Status{Available: true})

					synctest.Wait()
					require.NotNil(t, received, "internalmessage SoftRestart callback must fire")
					assert.Equal(t, expectedText, received.Text)
					assert.Equal(t, internalmessage.SensorMessageSoftRestart, received.Kind)
				}
			})
		})
	}
}

// TestCrdWatcherCallback_ConditionNotMet_DoesNotPublish verifies that the
// callback is a no-op when the callback condition is not satisfied.
func TestCrdWatcherCallback_ConditionNotMet_DoesNotPublish(t *testing.T) {
	tests := map[string]struct {
		pubsubEnabled bool
	}{
		"legacy": {pubsubEnabled: false},
		"pubsub": {pubsubEnabled: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), strconv.FormatBool(tc.pubsubEnabled))

				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				// No EXPECT() on mockDispatcher — any Publish call would fail the test.
				mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)
				pubSub := internalmessage.NewMessageSubscriber()

				var legacyReceived *internalmessage.SensorInternalMessage
				if !tc.pubsubEnabled {
					require.NoError(t, pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, func(msg *internalmessage.SensorInternalMessage) {
						legacyReceived = msg
					}))
				}

				cb := crdWatcherCallbackWrapper(
					context.Background(),
					allResourcesAvailable(), // expects Available == true
					pubSub,
					mockDispatcher,
					"should not fire",
				)
				cb(&watcher.Status{Available: false}) // condition not satisfied

				if !tc.pubsubEnabled {
					synctest.Wait()
					assert.Nil(t, legacyReceived, "legacy callback must not fire when condition not met")
				}
			})
		})
	}
}

// TestCrdWatcherCallback_ResourcesUnavailable verifies that the callback fires
// when using the resourcesUnavailable condition (i.e., when a CRD is removed).
func TestCrdWatcherCallback_ResourcesUnavailable(t *testing.T) {
	tests := map[string]struct {
		pubsubEnabled bool
	}{
		"legacy": {pubsubEnabled: false},
		"pubsub": {pubsubEnabled: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), strconv.FormatBool(tc.pubsubEnabled))

				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				const expectedText = "resources removed"
				mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)
				pubSub := internalmessage.NewMessageSubscriber()

				if tc.pubsubEnabled {
					var capturedEvent pubsub.Event
					mockDispatcher.EXPECT().Publish(gomock.Any()).DoAndReturn(func(e pubsub.Event) error {
						capturedEvent = e
						return nil
					})

					cb := crdWatcherCallbackWrapper(
						context.Background(),
						resourcesUnavailable(),
						pubSub,
						mockDispatcher,
						expectedText,
					)
					cb(&watcher.Status{Available: false})

					require.IsType(t, &events.SoftRestartEvent{}, capturedEvent)
					assert.Equal(t, expectedText, capturedEvent.(*events.SoftRestartEvent).Text)
				} else {
					var received *internalmessage.SensorInternalMessage
					require.NoError(t, pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, func(msg *internalmessage.SensorInternalMessage) {
						received = msg
					}))

					cb := crdWatcherCallbackWrapper(
						context.Background(),
						resourcesUnavailable(),
						pubSub,
						mockDispatcher,
						expectedText,
					)
					cb(&watcher.Status{Available: false})

					synctest.Wait()
					require.NotNil(t, received, "internalmessage callback must fire")
					assert.Equal(t, expectedText, received.Text)
					assert.Equal(t, internalmessage.SensorMessageSoftRestart, received.Kind)
				}
			})
		})
	}
}

// TestCrdWatcherCallback_ResourcesUnavailable_ConditionNotMet verifies that the
// unavailable condition does not fire when resources are available.
func TestCrdWatcherCallback_ResourcesUnavailable_ConditionNotMet(t *testing.T) {
	tests := map[string]struct {
		pubsubEnabled bool
	}{
		"legacy": {pubsubEnabled: false},
		"pubsub": {pubsubEnabled: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), strconv.FormatBool(tc.pubsubEnabled))

				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)
				pubSub := internalmessage.NewMessageSubscriber()

				var legacyReceived *internalmessage.SensorInternalMessage
				if !tc.pubsubEnabled {
					require.NoError(t, pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, func(msg *internalmessage.SensorInternalMessage) {
						legacyReceived = msg
					}))
				}

				cb := crdWatcherCallbackWrapper(
					context.Background(),
					resourcesUnavailable(),
					pubSub,
					mockDispatcher,
					"should not fire",
				)
				cb(&watcher.Status{Available: true}) // condition NOT met for resourcesUnavailable

				if !tc.pubsubEnabled {
					synctest.Wait()
					assert.Nil(t, legacyReceived, "legacy callback must not fire when condition not met")
				}
			})
		})
	}
}

// TestCrdWatcherCallback_CancelledContext verifies that when the context is
// cancelled before the CRD status fires, the event carries the cancelled
// context and reports IsExpired() == true.
func TestCrdWatcherCallback_CancelledContext(t *testing.T) {
	tests := map[string]struct {
		pubsubEnabled bool
	}{
		"legacy": {pubsubEnabled: false},
		"pubsub": {pubsubEnabled: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), strconv.FormatBool(tc.pubsubEnabled))

				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				mockDispatcher := listenerMocks.NewMockpubSubPublisher(mockCtrl)
				pubSub := internalmessage.NewMessageSubscriber()

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				if tc.pubsubEnabled {
					var capturedEvent pubsub.Event
					mockDispatcher.EXPECT().Publish(gomock.Any()).DoAndReturn(func(e pubsub.Event) error {
						capturedEvent = e
						return nil
					})

					cb := crdWatcherCallbackWrapper(
						ctx,
						allResourcesAvailable(),
						pubSub,
						mockDispatcher,
						"cancelled restart",
					)
					cb(&watcher.Status{Available: true})

					require.IsType(t, &events.SoftRestartEvent{}, capturedEvent)
					assert.True(t, capturedEvent.(*events.SoftRestartEvent).IsExpired(), "event must be expired when context is cancelled")
				} else {
					var received *internalmessage.SensorInternalMessage
					require.NoError(t, pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, func(msg *internalmessage.SensorInternalMessage) {
						received = msg
					}))

					cb := crdWatcherCallbackWrapper(
						ctx,
						allResourcesAvailable(),
						pubSub,
						mockDispatcher,
						"cancelled restart",
					)
					cb(&watcher.Status{Available: true})

					synctest.Wait()
					require.NotNil(t, received, "legacy callback must fire even with cancelled context")
					assert.True(t, received.IsExpired(), "legacy message must be expired when context is cancelled")
				}
			})
		})
	}
}
