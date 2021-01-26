package listener

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/client"
)

var (
	log = logging.LoggerForModule()
)

// SensorEventListener provides functionality to listen on sensor events.
type SensorEventListener interface {
	SensorEventStream() concurrency.ReadOnlyValueStream
}

// New returns a new kubernetes listener.
func New(client client.Interface, configHandler config.Handler, detector detector.Detector) common.SensorComponent {
	k := &listenerImpl{
		client:        client,
		eventsC:       make(chan *central.MsgFromSensor, 10),
		stopSig:       concurrency.NewSignal(),
		configHandler: configHandler,
		detector:      detector,
	}
	return k
}
