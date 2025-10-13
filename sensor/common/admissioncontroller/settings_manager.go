package admissioncontroller

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
)

// SettingsManager allows managing admission control settings. It allows updating policies and cluster configuration
// independently, and makes the settings available via a ValueStream.
//
//go:generate mockgen-wrapper
type SettingsManager interface {
	UpdatePolicies(allPolicies []*storage.Policy)
	UpdateConfig(config *storage.DynamicClusterConfig)
	UpdateResources(events ...*central.SensorEvent)
	GetResourcesForSync() []*sensor.AdmCtrlUpdateResourceRequest

	FlushCache()

	SettingsStream() concurrency.ReadOnlyValueStream[*sensor.AdmissionControlSettings]
	SensorEventsStream() concurrency.ReadOnlyValueStream[*sensor.AdmCtrlUpdateResourceRequest]
}
