package manager

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"

	admission "k8s.io/api/admission/v1beta1"
)

// Manager manages the main business logic of the admission control service.
type Manager interface {
	Start() error
	Stop()
	Stopped() concurrency.ErrorWaitable

	SettingsUpdateC() chan<- *sensor.AdmissionControlSettings
	SettingsStream() concurrency.ReadOnlyValueStream

	IsReady() bool

	HandleReview(review *admission.AdmissionRequest) (*admission.AdmissionResponse, error)
}

// New creates a new admission control manager
func New() Manager {
	return newManager()
}
