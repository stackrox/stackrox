package resolver

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

// New instantiates a Resolver component
func New(outputQueue component.OutputQueue, provider store.Provider, queueSize int) component.Resolver {
	return &resolverImpl{
		outputQueue:   outputQueue,
		innerQueue:    make(chan *component.ResourceEvent, queueSize),
		storeProvider: provider,
		stopSig:       concurrency.NewSignal(),
	}
}
