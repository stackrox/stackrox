package resolver

import (
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

// New instantiates a Resolver component
func New(outputQueue component.OutputQueue) component.Resolver {
	return &resolverImpl{
		outputQueue: outputQueue,
		innerQueue:  make(chan *component.ResourceEvent),
	}
}
