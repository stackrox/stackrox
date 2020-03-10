package admissioncontroller

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
)

// SettingsManager allows managing admission control settings. It allows updating policies and cluster configuration
// independently, and makes the settings available via a ValueStream.
type SettingsManager interface {
	UpdatePolicies(allPolicies []*storage.Policy)
	UpdateConfig(config *storage.DynamicClusterConfig)

	SettingsStream() concurrency.ReadOnlyValueStream
}
