package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// convertStorageClusterToAPI converts a storage.Cluster to the API representation.
// All fields are copied 1:1. The API type is structurally identical to the storage
// type today; they are separated to allow independent evolution in the future.
func convertStorageClusterToAPI(cluster *storage.Cluster) *v1.ClusterConfig {
	if cluster == nil {
		return nil
	}
	return &v1.ClusterConfig{
		Id:                             cluster.GetId(),
		Name:                           cluster.GetName(),
		Type:                           cluster.GetType(),
		Labels:                         cluster.GetLabels(),
		MainImage:                      cluster.GetMainImage(),
		CollectorImage:                 cluster.GetCollectorImage(),
		CentralApiEndpoint:             cluster.GetCentralApiEndpoint(),
		CollectionMethod:               cluster.GetCollectionMethod(),
		AdmissionController:            cluster.GetAdmissionController(),
		AdmissionControllerUpdates:     cluster.GetAdmissionControllerUpdates(),
		AdmissionControllerEvents:      cluster.GetAdmissionControllerEvents(),
		AdmissionControllerFailOnError: cluster.GetAdmissionControllerFailOnError(),
		DynamicConfig:                  cluster.GetDynamicConfig(),
		TolerationsConfig:              cluster.GetTolerationsConfig(),
		SlimCollector:                  cluster.GetSlimCollector(),
		Priority:                       cluster.GetPriority(),
		ManagedBy:                      cluster.GetManagedBy(),
		Status:                         cluster.GetStatus(),
		HealthStatus:                   cluster.GetHealthStatus(),
		HelmConfig:                     cluster.GetHelmConfig(),
		MostRecentSensorId:             cluster.GetMostRecentSensorId(),
		AuditLogState:                  cluster.GetAuditLogState(),
		InitBundleId:                   cluster.GetInitBundleId(),
		SensorCapabilities:             cluster.GetSensorCapabilities(),
	}
}

// convertStorageClustersToAPI converts a slice of storage.Cluster to API representations.
func convertStorageClustersToAPI(clusters []*storage.Cluster) []*v1.ClusterConfig {
	if clusters == nil {
		return nil
	}
	result := make([]*v1.ClusterConfig, len(clusters))
	for i, c := range clusters {
		result[i] = convertStorageClusterToAPI(c)
	}
	return result
}

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
