package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	clusterutil "github.com/stackrox/rox/pkg/cluster"
)

// convertStorageClusterToAPI delegates to the shared conversion in pkg/cluster.
var convertStorageClusterToAPI = clusterutil.StorageClusterToAPIClusterConfig

// convertStorageClustersToAPI delegates to the shared conversion in pkg/cluster.
var convertStorageClustersToAPI = clusterutil.StorageClustersToAPIClusterConfigs

// convertAPIClusterToStorage converts an API ClusterConfig to storage.Cluster for persistence.
//
// Server-managed fields (status, health_status, most_recent_sensor_id,
// sensor_capabilities, audit_log_state, helm_config, init_bundle_id) are
// intentionally NOT copied from the API request. If a client sends values
// for these fields, they are silently ignored. This preserves backward
// compatibility with older clients that may send the full object on update,
// while ensuring server-authoritative fields are never overwritten by clients.
// The server always populates these fields from its own state.
func convertAPIClusterToStorage(config *v1.ClusterConfig) *storage.Cluster {
	if config == nil {
		return nil
	}
	return &storage.Cluster{
		Id:                             config.GetId(),
		Name:                           config.GetName(),
		Type:                           config.GetType(),
		Labels:                         config.GetLabels(),
		MainImage:                      config.GetMainImage(),
		CollectorImage:                 config.GetCollectorImage(),
		CentralApiEndpoint:             config.GetCentralApiEndpoint(),
		CollectionMethod:               config.GetCollectionMethod(),
		AdmissionController:            config.GetAdmissionController(),
		AdmissionControllerUpdates:     config.GetAdmissionControllerUpdates(),
		AdmissionControllerEvents:      config.GetAdmissionControllerEvents(),
		AdmissionControllerFailOnError: config.GetAdmissionControllerFailOnError(),
		DynamicConfig:                  config.GetDynamicConfig(),
		TolerationsConfig:              config.GetTolerationsConfig(),
		SlimCollector:                  config.GetSlimCollector(),
		Priority:                       config.GetPriority(),
		ManagedBy:                      config.GetManagedBy(),
		// Server-managed fields below are intentionally omitted.
		// Values sent by clients for these fields are silently discarded:
		// - Status
		// - HealthStatus
		// - HelmConfig
		// - MostRecentSensorId
		// - AuditLogState
		// - InitBundleId
		// - SensorCapabilities
	}
}
