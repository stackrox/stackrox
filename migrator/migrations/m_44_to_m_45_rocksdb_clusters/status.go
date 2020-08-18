package m44tom45

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

const (
	healthySensorThreshold  = 1 * time.Minute
	degradedSensorThreshold = 3 * time.Minute
)

func populateSensorStatus(previousContact time.Time) storage.ClusterHealthStatus_HealthStatusLabel {
	// sensor never connected with central
	if previousContact.IsZero() {
		return storage.ClusterHealthStatus_UNINITIALIZED
	}

	diff := time.Since(previousContact)
	if diff <= healthySensorThreshold {
		return storage.ClusterHealthStatus_HEALTHY
	}
	if diff <= degradedSensorThreshold {
		return storage.ClusterHealthStatus_DEGRADED
	}
	return storage.ClusterHealthStatus_UNHEALTHY
}
