package connection

import (
	"github.com/stackrox/rox/central/scrape"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

// SensorConnection provides a handle to an established connection from a sensor.
type SensorConnection interface {
	Terminate(err error) bool
	Stopped() concurrency.ReadOnlyErrorSignal

	InjectMessage(msg *central.MsgToSensor) error

	Scrapes() scrape.Controller
}
