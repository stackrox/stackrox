package sensor

import (
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	roxsync "github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeNotifiable struct {
	received []common.SensorComponentEvent
}

func (f *fakeNotifiable) Notify(e common.SensorComponentEvent) {
	f.received = append(f.received, e)
}

func newLifecycleTestDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	disp, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
		lane.NewBlockingLane(pubsub.CentralReachableLane),
		lane.NewBlockingLane(pubsub.SensorOfflineLane),
		lane.NewBlockingLane(pubsub.HandshakeSyncFinishedLane),
	}))
	require.NoError(t, err)
	t.Cleanup(disp.Stop)
	return disp
}

// TestNotifyAllOnSignal_DualPublish verifies that, once `signal` fires, notifyAllOnSignal
// dual-publishes both lifecycle events to PubSub and drives the legacy Notify path in the
// documented order (CentralReachable fully delivered before HandshakeSyncFinished).
func TestNotifyAllOnSignal_DualPublish(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

		dispatcher := newLifecycleTestDispatcher(t)
		reachable := make(chan pubsub.Event, 1)
		syncFinished := make(chan pubsub.Event, 1)
		require.NoError(t, dispatcher.RegisterConsumerToLane(pubsub.DefaultConsumer,
			pubsub.CentralReachableTopic, pubsub.CentralReachableLane, func(e pubsub.Event) error {
				reachable <- e
				return nil
			}))
		require.NoError(t, dispatcher.RegisterConsumerToLane(pubsub.DefaultConsumer,
			pubsub.HandshakeSyncFinishedTopic, pubsub.HandshakeSyncFinishedLane, func(e pubsub.Event) error {
				syncFinished <- e
				return nil
			}))

		notifiable := &fakeNotifiable{}
		s := &Sensor{
			pubSubDispatcher: dispatcher,
			notifyList:       []common.Notifiable{notifiable},
			currentStateMtx:  &roxsync.Mutex{},
			stoppedSig:       concurrency.NewErrorSignal(),
		}
		fakeCC := &fakeCentralComm{stopped: concurrency.NewErrorSignal()}
		signal := concurrency.NewSignal()

		go s.notifyAllOnSignal(&signal, fakeCC)
		synctest.Wait()

		signal.Signal()
		synctest.Wait()

		select {
		case e := <-reachable:
			assert.IsType(t, &events.CentralReachableEvent{}, e)
		default:
			t.Fatal("CentralReachableEvent was not published")
		}
		select {
		case e := <-syncFinished:
			assert.IsType(t, &events.HandshakeSyncFinishedEvent{}, e)
		default:
			t.Fatal("HandshakeSyncFinishedEvent was not published")
		}
		assert.Equal(t, []common.SensorComponentEvent{
			common.SensorComponentEventCentralReachable,
			common.SensorComponentEventSyncFinished,
		}, notifiable.received, "legacy Notify path must fire CentralReachable before SyncFinished")
	})
}

func TestNotifyAllComponents_LegacyOnlyWhenPubSubDisabled(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "false")

	notifiable := &fakeNotifiable{}
	s := &Sensor{
		notifyList: []common.Notifiable{notifiable},
	}

	assert.NotPanics(t, func() {
		s.notifyAllComponents(
			common.SensorComponentEventCentralReachable,
			common.SensorComponentEventOfflineMode,
			common.SensorComponentEventSyncFinished,
		)
	})
	assert.Equal(t, []common.SensorComponentEvent{
		common.SensorComponentEventCentralReachable,
		common.SensorComponentEventOfflineMode,
		common.SensorComponentEventSyncFinished,
	}, notifiable.received)
}

// TestOfflineTransition_DedupesPublish verifies that publishLifecycleEvent is only invoked
// once per real state transition: repeated s.changeState(OfflineMode) calls while already
// offline must not re-publish, mirroring changeStateNoLock's existing dedup for legacy Notify.
func TestOfflineTransition_DedupesPublish(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

		dispatcher := newLifecycleTestDispatcher(t)
		var mu roxsync.Mutex
		publishCount := 0
		require.NoError(t, dispatcher.RegisterConsumerToLane(pubsub.DefaultConsumer,
			pubsub.SensorOfflineTopic, pubsub.SensorOfflineLane, func(e pubsub.Event) error {
				mu.Lock()
				defer mu.Unlock()
				publishCount++
				return nil
			}))

		notifiable := &fakeNotifiable{}
		s := &Sensor{
			pubSubDispatcher: dispatcher,
			notifyList:       []common.Notifiable{notifiable},
			currentStateMtx:  &roxsync.Mutex{},
		}

		goOffline := func() {
			if s.changeState(common.SensorComponentEventOfflineMode) {
				s.publishLifecycleEvent(&events.SensorOfflineEvent{})
			}
		}

		goOffline()
		synctest.Wait()
		goOffline()
		synctest.Wait()

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 1, publishCount, "must not re-publish on a repeated no-op transition")
		assert.Equal(t, []common.SensorComponentEvent{common.SensorComponentEventOfflineMode}, notifiable.received,
			"legacy Notify must also fire exactly once")
	})
}
