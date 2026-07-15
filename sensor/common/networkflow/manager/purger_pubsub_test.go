package manager

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newPurgerForPubSubTest builds a purger with a fast real ticker (not the
// WithPurgerTicker test override, which decouples purgerTickerC from
// purgerTicker -- Reset()ing the real ticker wouldn't be observable through
// an overridden channel), so that resetting purgerTicker via the PubSub
// callback is observable through purgingDone firing.
func newPurgerForPubSubTest(t *testing.T, dispatcher *capturingDispatcher) *NetworkFlowPurger {
	t.Helper()
	t.Setenv(env.EnrichmentPurgerTickerCycle.EnvVar(), "10ms")

	mockCtrl := gomock.NewController(t)
	enrichTickerC := make(chan time.Time)
	t.Cleanup(func() { close(enrichTickerC) })

	m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)
	return NewNetworkFlowPurger(mockEntityStore, time.Hour, dispatcher, WithManager(m))
}

// TestNewNetworkFlowPurger_PubSubEnabled_RegistersConsumer verifies that when
// the pubsub feature flag is enabled, Start() registers a consumer for
// ResourceSyncFinishedTopic.
func TestNewNetworkFlowPurger_PubSubEnabled_RegistersConsumer(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	capturing := &capturingDispatcher{}
	purger := newPurgerForPubSubTest(t, capturing)
	defer purger.Stop()

	require.NoError(t, purger.Start())

	require.NotNil(t, capturing.callback, "expected RegisterConsumerToLane to be called with a non-nil callback")
	assert.Equal(t, pubsub.NetworkFlowPurgerResourceSyncFinishedConsumer, capturing.consumerID)
	assert.Equal(t, pubsub.ResourceSyncFinishedTopic, capturing.topic)
	assert.Equal(t, pubsub.ResourceSyncFinishedLane, capturing.laneID)
}

// TestNewNetworkFlowPurger_PubSubEnabled_CallbackResetsTicker verifies the
// callback resets the purger ticker, mirroring the legacy Notify behavior
// exercised by TestPurgerStartWithTicker.
func TestNewNetworkFlowPurger_PubSubEnabled_CallbackResetsTicker(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	capturing := &capturingDispatcher{}
	purger := newPurgerForPubSubTest(t, capturing)
	defer purger.Stop()
	require.NoError(t, purger.Start())
	require.NotNil(t, capturing.callback)

	require.NoError(t, capturing.callback(&events.ResourceSyncFinishedEvent{}))

	assert.Eventually(t, purger.purgingDone.IsDone, 2*time.Second, 10*time.Millisecond,
		"purgingDone must be signaled once the ticker (reset by the callback) fires")
}

// TestNewNetworkFlowPurger_PubSubEnabled_CallbackSkipsExpiredEvent verifies
// that the callback drops stale events whose validity context has been
// cancelled, without resetting the ticker.
func TestNewNetworkFlowPurger_PubSubEnabled_CallbackSkipsExpiredEvent(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	capturing := &capturingDispatcher{}
	purger := newPurgerForPubSubTest(t, capturing)
	defer purger.Stop()
	require.NoError(t, purger.Start())
	require.NotNil(t, capturing.callback)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, capturing.callback(&events.ResourceSyncFinishedEvent{Validity: ctx}))

	assert.Never(t, purger.purgingDone.IsDone, 200*time.Millisecond, 10*time.Millisecond,
		"purgingDone must not be signaled for an expired event since the ticker isn't reset")
}

// TestNewNetworkFlowPurger_PubSubEnabled_CallbackRejectsWrongEventType
// verifies that the callback returns an error for an unexpected event type.
func TestNewNetworkFlowPurger_PubSubEnabled_CallbackRejectsWrongEventType(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	capturing := &capturingDispatcher{}
	purger := newPurgerForPubSubTest(t, capturing)
	defer purger.Stop()
	require.NoError(t, purger.Start())
	require.NotNil(t, capturing.callback)

	err := capturing.callback(&events.SoftRestartEvent{Text: "wrong type"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestNewNetworkFlowPurger_PubSubEnabled_RealDispatcher creates a real PubSub
// dispatcher (not a capturing mock), publishes a ResourceSyncFinishedEvent
// through it, and verifies purgingDone eventually signals -- exercising the
// actual lane routing end-to-end.
func TestNewNetworkFlowPurger_PubSubEnabled_RealDispatcher(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
	t.Setenv(env.EnrichmentPurgerTickerCycle.EnvVar(), "10ms")

	disp, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.ResourceSyncFinishedLane),
		},
	))
	require.NoError(t, err)
	defer disp.Stop()

	mockCtrl := gomock.NewController(t)
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	m, mockEntityStore, _, _ := createManager(mockCtrl, enrichTickerC)

	purger := NewNetworkFlowPurger(mockEntityStore, time.Hour, disp, WithManager(m))
	require.NoError(t, purger.Start())
	defer purger.Stop()

	require.NoError(t, disp.Publish(&events.ResourceSyncFinishedEvent{}))

	assert.Eventually(t, purger.purgingDone.IsDone, 2*time.Second, 10*time.Millisecond,
		"purgingDone must be signaled after publishing through the real dispatcher")
}
