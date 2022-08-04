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

	// HealthyAdmissionControllerThreshold represents the threshold for overall admission control status to be healthy.
	// This threshold is calculated as fraction of total desired admission control pods that are ready.
	HealthyAdmissionControllerThreshold = float64(1)
	// DegradedAdmissionControlThreshold represents the threshold for overall admission control status to be healthy.
	// This threshold is calculated as fraction of total desired admission control pods that are ready.
	DegradedAdmissionControlThreshold = float64(0.66)

	// healthyLocalScannerThreshold represents the threshold for overall local scanner status to be healthy.
	// This threshold is calculated as fraction of total desired local scanner pods that are ready.
	healthyLocalScannerThreshold = float64(1)
	// degradedLocalScannerThreshold represents the threshold for overall local scanner status to be degraded.
	// This threshold is calculated as fraction of total desired local scanner pods that are ready.
	degradedLocalScannerThreshold = float64(0.66)
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

// PopulateAdmissionControlStatus returns admission control status based on fraction of total desired admission control pods that
// have not failed to register with sensor.
func PopulateAdmissionControlStatus(admissionControlHealthInfo *storage.AdmissionControlHealthInfo) storage.ClusterHealthStatus_HealthStatusLabel {
	if admissionControlHealthInfo == nil {
		return storage.ClusterHealthStatus_UNINITIALIZED
	}

	if admissionControlHealthInfo.TotalDesiredPodsOpt == nil || admissionControlHealthInfo.TotalReadyPodsOpt == nil {
		// Fields will be nil if there was an error when trying to determine counts of desired/ready pods.
		// In this case we don't have enough information and can't report status as HEALTHY or even DEGRADED.
		// Reporting status as UNHEALTHY will attract user's attention to the problem and push them to resolve it.
		return storage.ClusterHealthStatus_UNHEALTHY
	}

	desiredPods := admissionControlHealthInfo.GetTotalDesiredPods()
	readyPods := admissionControlHealthInfo.GetTotalReadyPods()

	if desiredPods == 0 {
		return storage.ClusterHealthStatus_UNINITIALIZED
	}

	fraction := float64(readyPods) / float64(desiredPods)
	if fraction >= HealthyAdmissionControllerThreshold {
		return storage.ClusterHealthStatus_HEALTHY
	}
	if fraction >= DegradedAdmissionControlThreshold {
		return storage.ClusterHealthStatus_DEGRADED
	}
	return storage.ClusterHealthStatus_UNHEALTHY
}

// PopulateLocalScannerStatus returns local scanner status based on fraction of total desired pods that
// have not failed to register with sensor.
func PopulateLocalScannerStatus(localScannerHealthInfo *storage.ScannerHealthInfo) storage.ClusterHealthStatus_HealthStatusLabel {
	if localScannerHealthInfo == nil {
		return storage.ClusterHealthStatus_UNINITIALIZED
	}

	desiredPods := localScannerHealthInfo.GetTotalDesiredAnalyzerPods()
	readyPods := localScannerHealthInfo.GetTotalReadyAnalyzerPods()

	if desiredPods == 0 {
		return storage.ClusterHealthStatus_UNINITIALIZED
	}
	if localScannerHealthInfo.GetTotalReadyDbPods() == 0 {
		return storage.ClusterHealthStatus_UNHEALTHY
	}

	fraction := float64(readyPods) / float64(desiredPods)

	if fraction < degradedLocalScannerThreshold {
		return storage.ClusterHealthStatus_UNHEALTHY
	}
	if fraction < healthyLocalScannerThreshold {
		return storage.ClusterHealthStatus_DEGRADED
	}

	return storage.ClusterHealthStatus_HEALTHY
}

// PopulateOverallClusterStatus returns overall cluster status based on sensor status and collector status.
func PopulateOverallClusterStatus(clusterHealth *storage.ClusterHealthStatus) storage.ClusterHealthStatus_HealthStatusLabel {
	sensorStatus := clusterHealth.GetSensorHealthStatus()
	collectorStatus := clusterHealth.GetCollectorHealthStatus()
	admissionControlStatus := clusterHealth.GetAdmissionControlHealthStatus()

	// Collector having states other than default state when sensor is in default state is unlikely, but still check it first.
	if sensorStatus == storage.ClusterHealthStatus_UNINITIALIZED {
		return sensorStatus
	}

	if collectorStatus == storage.ClusterHealthStatus_UNHEALTHY ||
		admissionControlStatus == storage.ClusterHealthStatus_UNHEALTHY {
		return storage.ClusterHealthStatus_UNHEALTHY
	}

	if collectorStatus == storage.ClusterHealthStatus_DEGRADED || admissionControlStatus == storage.ClusterHealthStatus_DEGRADED {
		if sensorStatus == storage.ClusterHealthStatus_UNHEALTHY {
			return storage.ClusterHealthStatus_UNHEALTHY
		}
		return storage.ClusterHealthStatus_DEGRADED
	}

	// If we are here it means collector and admission controller is not unhealthy or degraded. Overall cluster health is determined by sensor status.
	return sensorStatus
}
