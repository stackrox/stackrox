package types

import "bitbucket.org/stack-rox/apollo/apollo/types"

// Listener is the interface that allows for propagation of events back from the orchestrators
type Listener interface {
	Events() <-chan types.Event
	GetContainers() ([]*types.Container, error)
	Start()
	Done()
}
