package types

import "bitbucket.org/stack-rox/apollo/apollo/types"

// Listener is the interface that allows for propagation of events back from the orchestrator.
type Listener interface {
	Events() <-chan types.DeploymentEvent
	Start()
	Stop()
}
