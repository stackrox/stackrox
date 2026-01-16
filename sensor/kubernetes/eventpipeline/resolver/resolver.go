package resolver

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dedupingqueue"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

type pubSubRegister interface {
	RegisterConsumerToLane(pubsub.Topic, pubsub.LaneID, pubsub.EventCallback) error
}

// New instantiates a Resolver component.
func New(outputQueue component.OutputQueue, provider store.Provider, queueSize int, pubsubDispatcher pubSubRegister) (component.Resolver, error) {
	res := &resolverImpl{
		outputQueue:           outputQueue,
		innerQueue:            make(chan *component.ResourceEvent, queueSize),
		storeProvider:         provider,
		stopper:               concurrency.NewStopper(),
		pullAndResolveStopped: concurrency.NewSignal(),
		deploymentRefQueue:    nil,
	}
	if features.SensorInternalPubSub.Enabled() {
		if pubsubDispatcher == nil {
			return nil, errors.Errorf("pubsub dispatcher is null and the feature flag %q is enabled", features.SensorInternalPubSub.EnvVar())
		}
		if err := pubsubDispatcher.RegisterConsumerToLane(pubsub.KubernetesDispatcherEventTopic, pubsub.KubernetesDispatcherEventLane, res.ProcessResourceEvent); err != nil {
			return nil, errors.Wrapf(err, "unable to register resolver as consumer of topic %q in lane %q", pubsub.KubernetesDispatcherEventTopic.String(), pubsub.KubernetesDispatcherEventLane.String())
		}
		if err := pubsubDispatcher.RegisterConsumerToLane(pubsub.FromCentralResolverEventTopic, pubsub.FromCentralResolverEventLane, res.ProcessResourceEvent); err != nil {
			return nil, errors.Wrapf(err, "unable to register resolver as consumer of topic %q in lane %q", pubsub.FromCentralResolverEventTopic.String(), pubsub.FromCentralResolverEventLane.String())
		}
	}
	if features.SensorAggregateDeploymentReferenceOptimization.Enabled() {
		res.deploymentRefQueue = dedupingqueue.NewDedupingQueue[string](
			dedupingqueue.WithSizeMetrics[string](metrics.ResolverDedupingQueueSize))
	}
	return res, nil
}
