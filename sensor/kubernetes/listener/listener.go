package listener

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
)

var (
	log = logging.LoggerForModule()
)

// New returns a new kubernetes listener.
func New(configHandler config.Handler, detector detector.Detector) common.SensorComponent {
	k := &listenerImpl{
		clients:       createClient(),
		eventsC:       make(chan *central.MsgFromSensor, 10),
		stopSig:       concurrency.NewSignal(),
		configHandler: configHandler,
		detector:      detector,
	}
	return k
}
