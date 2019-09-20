package listener

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/config"
)

var (
	log = logging.LoggerForModule()
)

// New returns a new kubernetes listener.
func New(configHandler config.Handler) listeners.Listener {
	k := &listenerImpl{
		clients:       createClient(),
		eventsC:       make(chan *central.SensorEvent, 10),
		stopSig:       concurrency.NewSignal(),
		configHandler: configHandler,
	}
	return k
}
