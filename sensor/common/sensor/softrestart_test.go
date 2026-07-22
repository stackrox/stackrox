package sensor

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	roxsync "github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
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

// ---- helper -----------------------------------------------------------------

func sensorForTest(t *testing.T, pubsubEnabled bool) (*Sensor, common.PubSubDispatcher, *internalmessage.MessageSubscriber) {
	t.Helper()
	s := &Sensor{
		centralCommunicationLock: &roxsync.Mutex{},
		pubSub:                   internalmessage.NewMessageSubscriber(),
	}

	var disp common.PubSubDispatcher
	if pubsubEnabled {
		var err error
		disp, err = pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
			[]pubsub.LaneConfig{
				lane.NewBlockingLane(pubsub.SoftRestartLane),
			},
		))
		require.NoError(t, err)
		t.Cleanup(disp.Stop)
		s.pubSubDispatcher = disp
		s.registerSoftRestartHandler()
	} else {
		require.NoError(t, s.pubSub.Subscribe(internalmessage.SensorMessageSoftRestart, s.onSoftRestartLegacy))
	}

	return s, disp, s.pubSub
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ---- tests ------------------------------------------------------------------

// TestSoftRestart_NilCommunication verifies that both paths gracefully handle
// when the central connection has not been established yet.
func TestSoftRestart_NilCommunication(t *testing.T) {
	tests := map[string]struct {
		pubsubEnabled bool
	}{
		"legacy": {pubsubEnabled: false},
		"pubsub": {pubsubEnabled: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), boolString(tc.pubsubEnabled))
				s, disp, msgSub := sensorForTest(t, tc.pubsubEnabled)

				if tc.pubsubEnabled {
					require.NoError(t, disp.Publish(&events.SoftRestartEvent{
						LifecycleEvent: events.LifecycleEvent{Text: "restart"},
					}))
				} else {
					require.NoError(t, msgSub.Publish(&internalmessage.SensorInternalMessage{
						Kind:     internalmessage.SensorMessageSoftRestart,
						Text:     "restart",
						Validity: context.Background(),
					}))
				}

				synctest.Wait()
				// Test passes if no panic occurs (nil communication is handled gracefully)
				assert.Nil(t, s.centralCommunication)
			})
		})
	}
}

// TestSoftRestart_StopsConnection verifies that both paths call Stop() on the
// active central communication when a soft restart event is received.
func TestSoftRestart_StopsConnection(t *testing.T) {
	tests := map[string]struct {
		pubsubEnabled bool
	}{
		"legacy": {pubsubEnabled: false},
		"pubsub": {pubsubEnabled: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), boolString(tc.pubsubEnabled))
				s, disp, msgSub := sensorForTest(t, tc.pubsubEnabled)
				fakeCC := &fakeCentralComm{}
				s.centralCommunication = fakeCC

				if tc.pubsubEnabled {
					require.NoError(t, disp.Publish(&events.SoftRestartEvent{
						LifecycleEvent: events.LifecycleEvent{Text: "restart"},
					}))
				} else {
					require.NoError(t, msgSub.Publish(&internalmessage.SensorInternalMessage{
						Kind:     internalmessage.SensorMessageSoftRestart,
						Text:     "restart",
						Validity: context.Background(),
					}))
				}

				synctest.Wait()
				assert.Equal(t, 1, fakeCC.stopCount, "Stop() must be called exactly once")
			})
		})
	}
}

// TestSoftRestart_SkipsExpiredEvent verifies that both paths ignore events
// whose validity context has been cancelled.
func TestSoftRestart_SkipsExpiredEvent(t *testing.T) {
	tests := map[string]struct {
		pubsubEnabled bool
	}{
		"legacy": {pubsubEnabled: false},
		"pubsub": {pubsubEnabled: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), boolString(tc.pubsubEnabled))
				s, disp, msgSub := sensorForTest(t, tc.pubsubEnabled)
				fakeCC := &fakeCentralComm{}
				s.centralCommunication = fakeCC

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				if tc.pubsubEnabled {
					require.NoError(t, disp.Publish(&events.SoftRestartEvent{
						LifecycleEvent: events.LifecycleEvent{
							Text:     "expired restart",
							Validity: ctx,
						},
					}))
				} else {
					require.NoError(t, msgSub.Publish(&internalmessage.SensorInternalMessage{
						Kind:     internalmessage.SensorMessageSoftRestart,
						Text:     "expired restart",
						Validity: ctx,
					}))
				}

				synctest.Wait()
				assert.Equal(t, 0, fakeCC.stopCount, "Stop() must not be called for expired event")
			})
		})
	}
}

// TestSoftRestart_PubSubRejectsWrongEventType verifies that the pubsub callback
// returns an error when it receives an unexpected event type. This is a handler
// unit test (not a delivery test), so it has no legacy equivalent.
func TestSoftRestart_PubSubRejectsWrongEventType(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
		s, _, _ := sensorForTest(t, true)
		fakeCC := &fakeCentralComm{}
		s.centralCommunication = fakeCC

		err := s.onSoftRestart(&events.ResourceSyncFinishedEvent{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected event type")
		assert.Equal(t, 0, fakeCC.stopCount, "Stop() must not be called for wrong event type")
	})
}
