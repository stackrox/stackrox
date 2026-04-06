package cluster

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// StorageClusterToAPIClusterConfig converts a storage.Cluster to the API representation.
// All fields are copied 1:1. The API type is structurally identical to the storage
// type today; they are separated to allow independent evolution in the future.
func StorageClusterToAPIClusterConfig(cluster *storage.Cluster) *v1.ClusterConfig {
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

// StorageClustersToAPIClusterConfigs converts a slice of storage.Cluster to API representations.
func StorageClustersToAPIClusterConfigs(clusters []*storage.Cluster) []*v1.ClusterConfig {
	if clusters == nil {
		return nil
	}
	result := make([]*v1.ClusterConfig, len(clusters))
	for i, c := range clusters {
		result[i] = StorageClusterToAPIClusterConfig(c)
	}
	return result
}
