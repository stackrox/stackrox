package clusterhealth

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

const (

	// HealthySensorThreshold represents the threshold for central-sensor disconnect after which sensor is considered as degraded.
	HealthySensorThreshold = 1 * time.Minute
	// DegradedSensorThreshold represents the threshold for central-sensor disconnect after which sensor is considered as unhealthy.
	DegradedSensorThreshold = 3 * time.Minute

	// HealthyCollectorThreshold represents the threshold for overall collector status to be healthy.
	// This threshold is calculated as fraction of total desired collector pods that are ready.
	HealthyCollectorThreshold = float64(1)
	// DegradedCollectorThreshold represents the threshold for overall collector status to be healthy.
	// This threshold is calculated as fraction of total desired collector pods that are ready.
	DegradedCollectorThreshold = float64(0.8)
)

// GetSensorStatus returns sensor status based on sensor's last contact with central
func GetSensorStatus(previousContact time.Time, newContact time.Time) storage.ClusterHealthStatus_HealthStatusLabel {
	// sensor never connected with central
	if previousContact.IsZero() && newContact.IsZero() {
		return storage.ClusterHealthStatus_UNINITIALIZED
	}

	// sensor has connected with central
	if !newContact.IsZero() {
		return storage.ClusterHealthStatus_HEALTHY
	}

	// sensor has lost connection with central
	newContact = time.Now()
	diff := time.Since(previousContact)
	if diff <= HealthySensorThreshold {
		return storage.ClusterHealthStatus_HEALTHY
	}
	if diff <= DegradedSensorThreshold {
		return storage.ClusterHealthStatus_DEGRADED
	}
	return storage.ClusterHealthStatus_UNHEALTHY
}

// GetCollectorStatus returns collector status based on fraction of total desired collector pods that have not failed to register with sensor.
func GetCollectorStatus(desiredPods int64, readyPods int64) storage.ClusterHealthStatus_HealthStatusLabel {
	if desiredPods == 0 {
		return storage.ClusterHealthStatus_UNINITIALIZED
	}

	fraction := float64(readyPods) / float64(desiredPods)
	if fraction >= HealthyCollectorThreshold {
		return storage.ClusterHealthStatus_HEALTHY
	}
	if fraction >= DegradedCollectorThreshold {
		return storage.ClusterHealthStatus_DEGRADED
	}
	return storage.ClusterHealthStatus_UNHEALTHY
}
