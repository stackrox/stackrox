package centralsensor

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

const (
	// SensorHelloMetadataKey is the key to indicate by both sensor and central that *sensor*, not *central*, will be
	// the first one to send a message on the stream. Sensor must not assume it can safely start by sending a message
	// unless it has received the header metadata from sensor, with the metadata value for that key set to true.
	SensorHelloMetadataKey = `Rox-Sensor-Hello`
)

// SecuredClusterIsNotManagedManually returns true if the cluster is not manually managed
// (i.e., it is managed by Helm or by the Operator)
func SecuredClusterIsNotManagedManually(helmManagedConfig *central.HelmManagedConfigInit) bool {
	return helmManagedConfig.GetManagedBy() != storage.ManagerType_MANAGER_TYPE_UNKNOWN &&
		helmManagedConfig.GetManagedBy() != storage.ManagerType_MANAGER_TYPE_MANUAL
}
