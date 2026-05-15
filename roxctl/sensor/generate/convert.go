package generate

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// clusterConfigToStorage converts a v1.ClusterConfig to storage.Cluster for use
// with validation functions that accept storage.Cluster.
func clusterConfigToStorage(config *v1.ClusterConfig) *storage.Cluster {
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
		HelmConfig:                     config.GetHelmConfig(),
	}
}
