package listeners

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// Creator is a function stub that defined how to create a Listener
type Creator func() (Listener, error)

// DeploymentEventWrap contains a DeploymentEvent and the original deployment event.
type DeploymentEventWrap struct {
	*v1.DeploymentEvent
	OriginalSpec interface{}
}

// Listener is the interface that allows for propagation of events back from the orchestrator.
type Listener interface {
	Events() <-chan *DeploymentEventWrap
	Start()
	Stop()
}
