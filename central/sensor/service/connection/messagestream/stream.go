package messagestream

import "github.com/stackrox/rox/generated/internalapi/central"

// CentralMessageStream is a stripped-down version of the SensorService Communicate stream, similar to SensorMessageStream in Sensor.
type CentralMessageStream interface {
	Send(msg *central.MsgToSensor) error
}
