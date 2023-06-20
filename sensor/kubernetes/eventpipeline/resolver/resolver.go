package resolver

import (
	"sync/atomic"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/uniqueue"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

// New instantiates a Resolver component
func New(outputQueue component.OutputQueue, provider store.Provider, queueSize int) component.Resolver {
	return &resolverImpl{
		stopper:        concurrency.NewStopper(),
		outputQueue:    outputQueue,
		innerQueue:     nil,
		innerQueueSize: queueSize,
		storeProvider:  provider,
		stopped:        &atomic.Bool{},
		queue:          uniqueue.NewUniQueue[deploymentRef](queueSize),
	}
}
