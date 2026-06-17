package sensor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/sensor/queue"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
)

func buildPubSubDispatcher(releaseBuild bool) (common.PubSubDispatcher, error) {
	laneType := lane.TypeConcurrent
	consumerType := consumer.TypeBuffered
	if !releaseBuild && !env.PubSubConcurrentLanes.BooleanSetting() {
		laneType = lane.TypeBlocking
		consumerType = consumer.TypeDefault
	} else if !releaseBuild {
		log.Info("PubSub concurrent lanes enabled via environment variable")
	}

	eventPipelineQueueSize := queue.ScaleSizeOnNonDefault(env.EventPipelineQueueSize)
	processIndicatorBufferSize := queue.ScaleSizeOnNonDefault(env.ProcessIndicatorBufferSize)

	laneSpecs := []lane.Spec{
		{ID: pubsub.KubernetesDispatcherEventLane, Type: laneType,
			Size: pointers.Int(eventPipelineQueueSize)},
		{ID: pubsub.FromCentralResolverEventLane, Type: laneType,
			Size: pointers.Int(eventPipelineQueueSize)},
		{ID: pubsub.EnrichedProcessIndicatorLane, Type: laneType,
			Consumer: &consumer.Spec{Type: consumerType, Size: pointers.Int(processIndicatorBufferSize)}},
		{ID: pubsub.UnenrichedProcessIndicatorLane, Type: laneType,
			Consumer: &consumer.Spec{Type: consumerType, Size: pointers.Int(processIndicatorBufferSize)}},
	}

	laneConfigs, err := lane.SpecsToConfigs(laneSpecs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create lane configurations")
	}

	return pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(laneConfigs))
}
