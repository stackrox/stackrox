package resolver

import (
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

// New instantiates a Resolver component
func New(outputQueue component.OutputQueue, deploymentStore store.DeploymentStore, provider store.Provider) component.Resolver {
	return &resolverImpl{
		outputQueue:     outputQueue,
		innerQueue:      make(chan *component.ResourceEvent),
		deploymentStore: deploymentStore,
		storeProvider:   provider,
	}
}
