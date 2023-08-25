package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetCluster provides a filled cluster object for testing purposes.
func GetCluster(name string) *storage.Cluster {
	return &storage.Cluster{
		Id:                         "",
		Name:                       name,
		Type:                       storage.ClusterType_KUBERNETES_CLUSTER,
		Labels:                     nil,
		MainImage:                  "quay.io/stackrox-io/main",
		CollectorImage:             "quay.io/stackrox-io/collector",
		CentralApiEndpoint:         "central.stackrox:443",
		RuntimeSupport:             true,
		CollectionMethod:           storage.CollectionMethod_EBPF,
		AdmissionController:        true,
		AdmissionControllerUpdates: false,
		AdmissionControllerEvents:  true,
		Status:                     nil,
		DynamicConfig: &storage.DynamicClusterConfig{
			AdmissionControllerConfig: &storage.AdmissionControllerConfig{
				Enabled:          false,
				TimeoutSeconds:   20,
				ScanInline:       false,
				DisableBypass:    false,
				EnforceOnUpdates: false,
			},
			RegistryOverride: "",
			DisableAuditLogs: true,
		},
		TolerationsConfig: nil,
		Priority:          0,
		HealthStatus:      nil,
		SlimCollector:     true,
		HelmConfig:        nil,
		MostRecentSensorId: &storage.SensorDeploymentIdentification{
			SystemNamespaceId:   "dbcbf202-6086-4bf9-8bc1-d10af3e36883",
			DefaultNamespaceId:  "fcab1a6d-07a3-4da9-a9cf-e286537ed4e3",
			AppNamespace:        "stackrox",
			AppNamespaceId:      "cd14a849-21d3-4351-9a56-8a066c2e83e1",
			AppServiceaccountId: "",
			K8SNodeName:         "colima",
		},
		AuditLogState: nil,
		InitBundleId:  "bb0e13e0-621a-4b2e-8fb9-af4e466763ff",
		ManagedBy:     storage.ManagerType_MANAGER_TYPE_MANUAL,
	}
}
