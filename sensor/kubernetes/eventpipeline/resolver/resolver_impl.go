package resolver

import (
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

type resolverImpl struct {
	outputQueue component.OutputQueue
	innerQueue  chan *component.ResourceEvent

	deploymentStore store.DeploymentStore
	storeProvider   store.Provider
}

// Start the resolverImpl component
func (r *resolverImpl) Start() error {
	go r.runResolver()
	return nil
}

// Stop the resolverImpl component
func (r *resolverImpl) Stop(_ error) {
	defer close(r.innerQueue)
}

// Send a ResourceEvent message to the inner queue
func (r *resolverImpl) Send(event *component.ResourceEvent) {
	r.innerQueue <- event
}

// runResolver reads messages from the inner queue and process the message
func (r *resolverImpl) runResolver() {
	for {
		msg, more := <-r.innerQueue
		if !more {
			return
		}
		r.processMessage(msg)
	}
}

// processMessage resolves the dependencies and forwards the message to the outputQueue
func (r *resolverImpl) processMessage(msg *component.ResourceEvent) {
	// TODO: resolve dependencies
	r.outputQueue.Send(msg)
}

var _ component.Resolver = (*resolverImpl)(nil)
