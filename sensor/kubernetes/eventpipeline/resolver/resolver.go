package resolver

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/uniqueue"
)

// New instantiates a Resolver component.
func New(outputQueue component.OutputQueue, provider store.Provider, queueSize int) component.Resolver {
	res := &resolverImpl{
		outputQueue:        outputQueue,
		innerQueue:         make(chan *component.ResourceEvent, queueSize),
		storeProvider:      provider,
		stopper:            concurrency.NewStopper(),
		deploymentRefQueue: nil,
	}
	if features.SensorAggregateDeploymentReferenceOptimization.Enabled() {
		res.deploymentRefQueue = uniqueue.NewUniQueue(uniqueue.WithMetrics(metrics.ResolverDedupingQueueSize))
	}
	return res
}
