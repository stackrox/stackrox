package sensor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/sensor/queue"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubConfig "github.com/stackrox/rox/sensor/common/pubsub/config"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
)

func buildPubSubDispatcher() (common.PubSubDispatcher, error) {
	laneType := pubsubConfig.LaneTypeConcurrent
	consumerType := pubsubConfig.ConsumerTypeBuffered
	if !buildinfo.ReleaseBuild && !env.PubSubConcurrentLanes.BooleanSetting() {
		laneType = pubsubConfig.LaneTypeBlocking
		consumerType = pubsubConfig.ConsumerTypeDefault
	}

	eventPipelineQueueSize := queue.ScaleSizeOnNonDefault(env.EventPipelineQueueSize)
	processIndicatorBufferSize := queue.ScaleSizeOnNonDefault(env.ProcessIndicatorBufferSize)

	laneSpecs := []pubsubConfig.LaneSpec{
		{ID: pubsub.KubernetesDispatcherEventLane, Type: laneType,
			Size: pointers.Int(eventPipelineQueueSize)},
		{ID: pubsub.FromCentralResolverEventLane, Type: laneType,
			Size: pointers.Int(eventPipelineQueueSize)},
		{ID: pubsub.EnrichedProcessIndicatorLane, Type: laneType,
			Consumer: &pubsubConfig.ConsumerSpec{Type: consumerType, Size: pointers.Int(processIndicatorBufferSize)}},
		{ID: pubsub.UnenrichedProcessIndicatorLane, Type: laneType,
			Consumer: &pubsubConfig.ConsumerSpec{Type: consumerType, Size: pointers.Int(processIndicatorBufferSize)}},
	}

	laneConfigs, err := pubsubConfig.SpecsToConfigs(laneSpecs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create lane configurations")
	}

	return pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(laneConfigs))
}
