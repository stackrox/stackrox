package listeners

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var logger = logging.LoggerForModule()

// Creator is a function stub that defined how to create a Listener
type Creator func() (Listener, error)

// Listener is the interface that allows for propagation of events back from the orchestrator.
type Listener interface {
	Events() <-chan *v1.SensorEvent
	Start()
	Stop()
}
