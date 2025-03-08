package messagestream

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// SensorMessageStream is a stripped-down version of the SensorService Communicate stream.
//
//go:generate mockgen-wrapper
type SensorMessageStream interface {
	Send(msg *central.MsgFromSensor) error
}
