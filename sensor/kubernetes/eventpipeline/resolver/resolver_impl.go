package resolver

import (
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

type resolverImpl struct {
	outputQueue component.OutputQueue
	innerQueue  chan *component.ResourceEvent
}

func (r *resolverImpl) Start() error {
	go r.runResolver()
	return nil
}

func (r *resolverImpl) Stop(_ error) {
	defer close(r.innerQueue)
}

func (r *resolverImpl) Send(event *component.ResourceEvent) {
	r.innerQueue <- event
}

func (r *resolverImpl) runResolver() {
	for {
		msg, more := <-r.innerQueue
		if !more {
			return
		}
		r.processMessage(msg)
	}
}

func (r *resolverImpl) processMessage(msg *component.ResourceEvent) {
	// TODO: resolve dependencies
	r.outputQueue.Send(msg)
}

var _ component.Resolver = (*resolverImpl)(nil)
