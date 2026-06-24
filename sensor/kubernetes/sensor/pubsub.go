package sensor

import (
	"math"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sensor/queue"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
)

// buildConcurrentLane creates a ConcurrentLane with a BufferedConsumer.
// When the buffer size is 0 (default unbounded), uses math.MaxInt to
// match the legacy unbounded queue behavior. When a size is explicitly
// configured, uses that size and drops when full.
func buildConcurrentLane(id pubsub.LaneID, bufferEnv *env.IntegerSetting) pubsub.LaneConfig {
	bufferSize := queue.ScaleSizeOnNonDefault(bufferEnv)
	if bufferSize <= 0 {
		log.Infof("PubSub lane %q: buffer size not configured, defaulting to math.MaxInt", id)
		bufferSize = math.MaxInt
	}
	return lane.NewConcurrentLane(id,
		lane.WithConcurrentLaneConsumer(
			consumer.NewBufferedConsumer(
				consumer.WithBufferedConsumerSize(bufferSize),
			),
		),
	)
}

func buildPubSubDispatcher() (common.PubSubDispatcher, error) {
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.KubernetesDispatcherEventLane),
			lane.NewBlockingLane(pubsub.FromCentralResolverEventLane),
			lane.NewBlockingLane(pubsub.EnrichedProcessIndicatorLane),
			lane.NewBlockingLane(pubsub.UnenrichedProcessIndicatorLane),
			buildConcurrentLane(pubsub.DetectorProcessIndicatorLane, env.DetectorProcessIndicatorBufferSize),
			buildConcurrentLane(pubsub.DetectorNetworkFlowLane, env.DetectorNetworkFlowBufferSize),
			buildConcurrentLane(pubsub.DetectorFileAccessLane, env.DetectorFileAccessBufferSize),
		},
	))
	if err != nil {
		return nil, errors.Wrap(err, "unable to create the pubsub dispatcher")
	}
	return dispatcher, nil
}
