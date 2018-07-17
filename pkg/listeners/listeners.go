package listeners

import (
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var logger = logging.LoggerForModule()

// Creator is a function stub that defined how to create a Listener
type Creator func() (Listener, error)

// Listener is the interface that allows for propagation of events back from the orchestrator.
type Listener interface {
	Events() <-chan *EventWrap
	Start()
	Stop()
}
