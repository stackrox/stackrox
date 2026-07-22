package manager

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/events"
	mocksExternalSrc "github.com/stackrox/rox/sensor/common/externalsrcs/mocks"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	mocksManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	"github.com/stackrox/rox/sensor/common/networkflow/updatecomputer"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newManagerForTest(t *testing.T, pubsubEnabled bool) (Manager, *networkFlowManager, common.PubSubDispatcher, *internalmessage.MessageSubscriber) {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	mockEntityStore := mocksManager.NewMockEntityStore(mockCtrl)
	mockExternalStore := mocksExternalSrc.NewMockStore(mockCtrl)
	mockDetector := mocksDetector.NewMockDetector(mockCtrl)

	msgSub := internalmessage.NewMessageSubscriber()

	var disp common.PubSubDispatcher
	if pubsubEnabled {
		var err error
		disp, err = pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
			[]pubsub.LaneConfig{
				lane.NewBlockingLane(pubsub.ResourceSyncFinishedLane),
			},
		))
		require.NoError(t, err)
		t.Cleanup(disp.Stop)
	}

	m := NewManager(
		mockEntityStore,
		mockExternalStore,
		mockDetector,
		msgSub,
		disp,
		updatecomputer.New(),
	)
	return m, m.(*networkFlowManager), disp, msgSub
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// TestResourceSyncFinished_MarksInitialSync verifies that both delivery paths
// (pubsub and legacy) correctly mark the manager's initialSync flag when the
// ResourceSyncFinished event is published.
func TestResourceSyncFinished_MarksInitialSync(t *testing.T) {
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
				_, mgr, disp, msgSub := newManagerForTest(t, tc.pubsubEnabled)

				assert.False(t, mgr.initialSync.Load(), "initialSync must be false before event")

				if tc.pubsubEnabled {
					require.NoError(t, disp.Publish(&events.ResourceSyncFinishedEvent{}))
				} else {
					require.NoError(t, msgSub.Publish(&internalmessage.SensorInternalMessage{
						Kind:     internalmessage.SensorMessageResourceSyncFinished,
						Text:     "test sync",
						Validity: context.Background(),
					}))
				}

				synctest.Wait()
				assert.True(t, mgr.initialSync.Load(), "initialSync must be true after event")
			})
		})
	}
}

// TestResourceSyncFinished_PubSubHonorsStopper verifies that when the manager's stopper
// has been triggered, the pubsub path exits early without marking initialSync.
// The legacy path doesn't check the stopper (it goes through Notify which doesn't guard on stopper).
func TestResourceSyncFinished_PubSubHonorsStopper(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
		_, mgr, disp, _ := newManagerForTest(t, true)

		mgr.stopper.Client().Stop()

		require.NoError(t, disp.Publish(&events.ResourceSyncFinishedEvent{}))

		synctest.Wait()
		assert.False(t, mgr.initialSync.Load(), "must not set initialSync when stopper is triggered")
	})
}

// TestResourceSyncFinished_SkipsExpiredEvent verifies that both paths drop stale
// events whose validity context has been cancelled.
func TestResourceSyncFinished_SkipsExpiredEvent(t *testing.T) {
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
				_, mgr, disp, msgSub := newManagerForTest(t, tc.pubsubEnabled)

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				if tc.pubsubEnabled {
					require.NoError(t, disp.Publish(&events.ResourceSyncFinishedEvent{
						LifecycleEvent: events.LifecycleEvent{Validity: ctx},
					}))
				} else {
					require.NoError(t, msgSub.Publish(&internalmessage.SensorInternalMessage{
						Kind:     internalmessage.SensorMessageResourceSyncFinished,
						Text:     "test sync",
						Validity: ctx,
					}))
				}

				synctest.Wait()
				assert.False(t, mgr.initialSync.Load(), "must not set initialSync for expired event")
			})
		})
	}
}

// TestResourceSyncFinished_ConcurrentDelivery fires events from many goroutines
// simultaneously and verifies no data races occur. Run with -race.
func TestResourceSyncFinished_ConcurrentDelivery(t *testing.T) {
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
				_, mgr, disp, msgSub := newManagerForTest(t, tc.pubsubEnabled)

				const goroutines = 50
				var wg sync.WaitGroup
				wg.Add(goroutines)
				for range goroutines {
					go func() {
						defer wg.Done()
						if tc.pubsubEnabled {
							_ = disp.Publish(&events.ResourceSyncFinishedEvent{})
						} else {
							_ = msgSub.Publish(&internalmessage.SensorInternalMessage{
								Kind:     internalmessage.SensorMessageResourceSyncFinished,
								Validity: context.Background(),
							})
						}
					}()
				}
				wg.Wait()
				synctest.Wait()

				assert.True(t, mgr.initialSync.Load(), "initialSync must be true after concurrent delivery")
			})
		})
	}
}

// TestResourceSyncFinished_PubSubRejectsWrongEventType verifies that the pubsub
// callback returns an error when it receives an unexpected event type. This is a
// handler unit test (not a delivery test), so it has no legacy equivalent.
func TestResourceSyncFinished_PubSubRejectsWrongEventType(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
		_, mgr, _, _ := newManagerForTest(t, true)

		err := mgr.handleResourceSyncEvent(&events.SoftRestartEvent{
			LifecycleEvent: events.LifecycleEvent{Text: "wrong type"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected event type")
		assert.False(t, mgr.initialSync.Load(), "must not set initialSync for wrong event type")
	})
}
