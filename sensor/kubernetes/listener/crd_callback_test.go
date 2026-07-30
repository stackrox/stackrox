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

type crdTestFixture struct {
	mockDispatcher *listenerMocks.MockpubSubPublisher
	pubSub         *internalmessage.MessageSubscriber
	capturedEvent  pubsub.Event
	legacyReceived *internalmessage.SensorInternalMessage
}

func newCrdTestFixture(t *testing.T, pubsubEnabled bool) *crdTestFixture {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)

	f := &crdTestFixture{
		mockDispatcher: listenerMocks.NewMockpubSubPublisher(mockCtrl),
		pubSub:         internalmessage.NewMessageSubscriber(),
	}

	if pubsubEnabled {
		f.mockDispatcher.EXPECT().Publish(gomock.Any()).DoAndReturn(func(e pubsub.Event) error {
			f.capturedEvent = e
			return nil
		})
	} else {
		require.NoError(t, f.pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, func(msg *internalmessage.SensorInternalMessage) {
			f.legacyReceived = msg
		}))
	}

	return f
}

func newCrdTestFixtureNoExpect(t *testing.T, pubsubEnabled bool) *crdTestFixture {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)

	f := &crdTestFixture{
		mockDispatcher: listenerMocks.NewMockpubSubPublisher(mockCtrl),
		pubSub:         internalmessage.NewMessageSubscriber(),
	}

	if !pubsubEnabled {
		require.NoError(t, f.pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, func(msg *internalmessage.SensorInternalMessage) {
			f.legacyReceived = msg
		}))
	}

	return f
}

func (f *crdTestFixture) invokeCallback(ctx context.Context, cond callbackCondition, text string, status *watcher.Status) {
	cb := crdWatcherCallbackWrapper(ctx, cond, f.pubSub, f.mockDispatcher, text)
	cb(status)
}

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

				const expectedText = "test soft restart"
				f := newCrdTestFixture(t, tc.pubsubEnabled)
				f.invokeCallback(context.Background(), allResourcesAvailable(), expectedText, &watcher.Status{Available: true})

				if tc.pubsubEnabled {
					require.IsType(t, &events.SoftRestartEvent{}, f.capturedEvent)
					assert.Equal(t, expectedText, f.capturedEvent.(*events.SoftRestartEvent).Text)
				} else {
					synctest.Wait()
					require.NotNil(t, f.legacyReceived, "internalmessage SoftRestart callback must fire")
					assert.Equal(t, expectedText, f.legacyReceived.Text)
					assert.Equal(t, internalmessage.SensorMessageSoftRestart, f.legacyReceived.Kind)
				}
			})
		})
	}
}

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

				f := newCrdTestFixtureNoExpect(t, tc.pubsubEnabled)
				f.invokeCallback(context.Background(), allResourcesAvailable(), "should not fire", &watcher.Status{Available: false})

				if !tc.pubsubEnabled {
					synctest.Wait()
					assert.Nil(t, f.legacyReceived, "legacy callback must not fire when condition not met")
				}
			})
		})
	}
}

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

				const expectedText = "resources removed"
				f := newCrdTestFixture(t, tc.pubsubEnabled)
				f.invokeCallback(context.Background(), resourcesUnavailable(), expectedText, &watcher.Status{Available: false})

				if tc.pubsubEnabled {
					require.IsType(t, &events.SoftRestartEvent{}, f.capturedEvent)
					assert.Equal(t, expectedText, f.capturedEvent.(*events.SoftRestartEvent).Text)
				} else {
					synctest.Wait()
					require.NotNil(t, f.legacyReceived, "internalmessage callback must fire")
					assert.Equal(t, expectedText, f.legacyReceived.Text)
					assert.Equal(t, internalmessage.SensorMessageSoftRestart, f.legacyReceived.Kind)
				}
			})
		})
	}
}

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

				f := newCrdTestFixtureNoExpect(t, tc.pubsubEnabled)
				f.invokeCallback(context.Background(), resourcesUnavailable(), "should not fire", &watcher.Status{Available: true})

				if !tc.pubsubEnabled {
					synctest.Wait()
					assert.Nil(t, f.legacyReceived, "legacy callback must not fire when condition not met")
				}
			})
		})
	}
}

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

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				f := newCrdTestFixture(t, tc.pubsubEnabled)
				f.invokeCallback(ctx, allResourcesAvailable(), "cancelled restart", &watcher.Status{Available: true})

				if tc.pubsubEnabled {
					require.IsType(t, &events.SoftRestartEvent{}, f.capturedEvent)
					assert.True(t, f.capturedEvent.(*events.SoftRestartEvent).IsExpired(), "event must be expired when context is cancelled")
				} else {
					synctest.Wait()
					require.NotNil(t, f.legacyReceived, "legacy callback must fire even with cancelled context")
					assert.True(t, f.legacyReceived.IsExpired(), "legacy message must be expired when context is cancelled")
				}
			})
		})
	}
}
