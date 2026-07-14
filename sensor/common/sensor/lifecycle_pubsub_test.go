package sensor

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/features"
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
		lane.NewBlockingLane(pubsub.SensorOnlineLane),
		lane.NewBlockingLane(pubsub.SensorOfflineLane),
		lane.NewBlockingLane(pubsub.SyncFinishedLane),
	}))
	require.NoError(t, err)
	t.Cleanup(disp.Stop)
	return disp
}

func TestNotifyAllComponents_DualPublish(t *testing.T) {
	cases := map[string]struct {
		notification common.SensorComponentEvent
		topic        pubsub.Topic
		laneID       pubsub.LaneID
		wantEvent    pubsub.Event
	}{
		"CentralReachable publishes SensorOnlineEvent": {
			notification: common.SensorComponentEventCentralReachable,
			topic:        pubsub.SensorOnlineTopic,
			laneID:       pubsub.SensorOnlineLane,
			wantEvent:    &events.SensorOnlineEvent{},
		},
		"OfflineMode publishes SensorOfflineEvent": {
			notification: common.SensorComponentEventOfflineMode,
			topic:        pubsub.SensorOfflineTopic,
			laneID:       pubsub.SensorOfflineLane,
			wantEvent:    &events.SensorOfflineEvent{},
		},
		"SyncFinished publishes SyncFinishedEvent": {
			notification: common.SensorComponentEventSyncFinished,
			topic:        pubsub.SyncFinishedTopic,
			laneID:       pubsub.SyncFinishedLane,
			wantEvent:    &events.SyncFinishedEvent{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

			dispatcher := newLifecycleTestDispatcher(t)
			received := make(chan pubsub.Event, 1)
			require.NoError(t, dispatcher.RegisterConsumerToLane(pubsub.DefaultConsumer, tc.topic, tc.laneID, func(e pubsub.Event) error {
				received <- e
				return nil
			}))

			notifiable := &fakeNotifiable{}
			s := &Sensor{
				pubSubDispatcher: dispatcher,
				notifyList:       []common.Notifiable{notifiable},
			}

			s.notifyAllComponents(tc.notification)

			select {
			case e := <-received:
				assert.IsType(t, tc.wantEvent, e)
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for published PubSub event")
			}
			assert.Equal(t, []common.SensorComponentEvent{tc.notification}, notifiable.received,
				"legacy Notify path must still fire alongside the PubSub publish (dual-path)")
		})
	}
}

func TestNotifyAllComponents_LegacyOnlyWhenPubSubDisabled(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "false")

	notifiable := &fakeNotifiable{}
	// pubSubDispatcher is intentionally left nil: if notifyAllComponents attempted
	// a publish with the flag disabled, this would panic.
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

func TestNotifyAllComponents_NoPubSubTopicForCentralReachableHTTP(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newLifecycleTestDispatcher(t)
	notifiable := &fakeNotifiable{}
	s := &Sensor{
		pubSubDispatcher: dispatcher,
		notifyList:       []common.Notifiable{notifiable},
	}

	// CentralReachableHTTP has no PubSub topic; publishLifecycleEvent must no-op
	// rather than error or panic trying to publish an unmapped event.
	assert.NotPanics(t, func() {
		s.notifyAllComponents(common.SensorComponentEventCentralReachableHTTP)
	})
	assert.Equal(t, []common.SensorComponentEvent{common.SensorComponentEventCentralReachableHTTP}, notifiable.received)
}
