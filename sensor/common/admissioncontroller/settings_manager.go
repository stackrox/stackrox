package admissioncontroller

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
)

// SettingsManager allows managing admission control settings. It allows updating policies and cluster configuration
// independently, and makes the settings available via a ValueStream.
//go:generate mockgen-wrapper
type SettingsManager interface {
	UpdatePolicies(allPolicies []*storage.Policy)
	UpdateConfig(config *storage.DynamicClusterConfig)
	UpdateResources(events ...*central.SensorEvent)
	GetResourcesForSync() []*sensor.AdmCtrlUpdateResourceRequest

	FlushCache()

	SettingsStream() concurrency.ReadOnlyValueStream
	SensorEventsStream() concurrency.ReadOnlyValueStream
}
