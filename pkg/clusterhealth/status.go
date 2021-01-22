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

// PopulateInactiveSensorStatus returns sensor status based on sensor's last contact with central in situation when there's no active connection between sensor and central.
func PopulateInactiveSensorStatus(lastContact time.Time) storage.ClusterHealthStatus_HealthStatusLabel {
	// sensor never connected with central
	if lastContact.IsZero() {
		return storage.ClusterHealthStatus_UNINITIALIZED
	}

	diff := time.Since(lastContact)
	if diff <= HealthySensorThreshold {
		return storage.ClusterHealthStatus_HEALTHY
	}
	if diff <= DegradedSensorThreshold {
		return storage.ClusterHealthStatus_DEGRADED
	}
	return storage.ClusterHealthStatus_UNHEALTHY
}

// PopulateCollectorStatus returns collector status based on fraction of total desired collector pods that have not failed to register with sensor.
func PopulateCollectorStatus(collectorInfo *storage.CollectorHealthInfo) storage.ClusterHealthStatus_HealthStatusLabel {
	if collectorInfo == nil {
		return storage.ClusterHealthStatus_UNINITIALIZED
	}

	if collectorInfo.TotalDesiredPodsOpt == nil || collectorInfo.TotalReadyPodsOpt == nil {
		// Fields will be nil if there was an error when trying to determine counts of desired/ready pods.
		// In this case we don't have enough information and can't report status as HEALTHY or even DEGRADED.
		// Reporting status as UNHEALTHY will attract user's attention to the problem and push them to resolve it.
		return storage.ClusterHealthStatus_UNHEALTHY
	}

	desiredPods := collectorInfo.GetTotalDesiredPods()
	readyPods := collectorInfo.GetTotalReadyPods()

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

// PopulateOverallClusterStatus returns overall cluster status based on sensor status and collector status.
func PopulateOverallClusterStatus(clusterHealth *storage.ClusterHealthStatus) storage.ClusterHealthStatus_HealthStatusLabel {
	sensorStatus := clusterHealth.GetSensorHealthStatus()
	collectorStatus := clusterHealth.GetCollectorHealthStatus()

	// Collector having states other than default state when sensor is in default state is unlikely, but still check it first.
	if sensorStatus == storage.ClusterHealthStatus_UNINITIALIZED {
		return sensorStatus
	}

	switch collectorStatus {
	case storage.ClusterHealthStatus_UNHEALTHY:
		return collectorStatus
	case storage.ClusterHealthStatus_DEGRADED:
		if sensorStatus == storage.ClusterHealthStatus_UNHEALTHY {
			return sensorStatus
		}
		return collectorStatus
	}
	// If we are here it means collector is not unhealthy or degraded. Overall cluster health is determined by sensor status.
	return sensorStatus
}
