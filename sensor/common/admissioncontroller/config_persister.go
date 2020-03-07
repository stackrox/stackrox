package admissioncontroller

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common"
)

// ConfigPersister is an interface for persisting the dynamic admission controller config as well as the policies it
// should enforce on.
type ConfigPersister interface {
	common.SensorComponent

	UpdatePolicies(allPolicies []*storage.Policy)
	UpdateConfig(config *storage.AdmissionControllerConfig)
}
