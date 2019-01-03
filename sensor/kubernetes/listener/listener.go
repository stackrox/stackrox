package listener

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// New returns a new kubernetes listener.
func New() listeners.Listener {
	k := &listenerImpl{
		clients: createClient(),
		eventsC: make(chan *central.SensorEvent, 10),
		stopSig: concurrency.NewSignal(),
	}
	return k
}
