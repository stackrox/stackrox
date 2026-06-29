package sensor

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEvent struct {
	topic pubsub.Topic
	lane  pubsub.LaneID
}

func (e *testEvent) Topic() pubsub.Topic { return e.topic }
func (e *testEvent) Lane() pubsub.LaneID { return e.lane }

func TestBuildPubSubDispatcher_CreatesWorkingDispatcher(t *testing.T) {
	disp, err := buildPubSubDispatcher()
	require.NoError(t, err)
	defer disp.Stop()

	tests := map[string]struct {
		topic pubsub.Topic
		lane  pubsub.LaneID
	}{
		"SoftRestart":          {topic: pubsub.SoftRestartTopic, lane: pubsub.SoftRestartLane},
		"ResourceSyncFinished": {topic: pubsub.ResourceSyncFinishedTopic, lane: pubsub.ResourceSyncFinishedLane},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var called atomic.Bool
			consumerID := pubsub.ConsumerID(100 + tc.lane)
			require.NoError(t, disp.RegisterConsumerToLane(consumerID, tc.topic, tc.lane, func(_ pubsub.Event) error {
				called.Store(true)
				return nil
			}))
			require.NoError(t, disp.Publish(&testEvent{topic: tc.topic, lane: tc.lane}))
			assert.Eventually(t, called.Load, 500*time.Millisecond, 5*time.Millisecond, "consumer must receive the event")
		})
	}
}
