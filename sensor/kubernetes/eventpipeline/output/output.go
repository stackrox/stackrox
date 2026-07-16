package output

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

type pubSubRegister interface {
	RegisterConsumerToLane(pubsub.ConsumerID, pubsub.Topic, pubsub.LaneID, pubsub.EventCallback) error
}

// New instantiates an output Queue component.
func New(detector detector.Detector, queueSize int, dispatcher pubSubRegister) (component.OutputQueue, error) {
	ch := make(chan *component.ResourceEvent, queueSize)
	forwardQueue := make(chan *message.ExpiringMessage, queueSize)
	outputQueue := &outputQueueImpl{
		detector:     detector,
		innerQueue:   ch,
		forwardQueue: forwardQueue,
		stopper:      concurrency.NewStopper(),
	}
	if features.SensorInternalPubSub.Enabled() {
		if dispatcher == nil {
			return nil, errors.Errorf("pubsub dispatcher is nil and the feature flag %q is enabled", features.SensorInternalPubSub.EnvVar())
		}
		if err := dispatcher.RegisterConsumerToLane(pubsub.OutputQueueConsumer, pubsub.ResolvedResourceEventTopic, pubsub.ResolvedResourceEventLane, outputQueue.ProcessResourceEvent); err != nil {
			return nil, errors.Wrapf(err, "unable to register output queue as consumer of topic %q in lane %q", pubsub.ResolvedResourceEventTopic.String(), pubsub.ResolvedResourceEventLane.String())
		}
	}
	return outputQueue, nil
}
