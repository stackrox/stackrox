package component

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/sensor/common"
)

// Pipeline defines the way to process a process signal
type Pipeline interface {
	Process(signal *sensor.ProcessSignal)
	Shutdown()
	Notify(e common.SensorComponentEvent)
}
