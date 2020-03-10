package manager

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Manager manages the main business logic of the admission control service.
type Manager interface {
	Start() error
	Stop()
	Stopped() concurrency.ErrorWaitable

	SettingsUpdateC() chan<- *sensor.AdmissionControlSettings
	SettingsStream() concurrency.ReadOnlyValueStream

	IsReady() bool
}

// New creates a new admission control manager
func New() Manager {
	return newManager()
}
