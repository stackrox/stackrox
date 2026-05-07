package cluster

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestStorageClusterToAPIClusterConfig_Nil(t *testing.T) {
	assert.Nil(t, StorageClusterToAPIClusterConfig(nil))
}

func TestStorageClustersToAPIClusterConfigs_Nil(t *testing.T) {
	assert.Nil(t, StorageClustersToAPIClusterConfigs(nil))
}

func TestStorageClustersToAPIClusterConfigs_Empty(t *testing.T) {
	result := StorageClustersToAPIClusterConfigs([]*storage.Cluster{})
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestStorageClusterToAPIClusterConfig_AllFields(t *testing.T) {
	cluster := &storage.Cluster{
		Id:                             "test-id",
		Name:                           "test-cluster",
		Type:                           storage.ClusterType_OPENSHIFT4_CLUSTER,
		Labels:                         map[string]string{"env": "prod"},
		MainImage:                      "quay.io/stackrox/main:latest",
		CollectorImage:                 "quay.io/stackrox/collector:latest",
		CentralApiEndpoint:             "central.stackrox:443",
		CollectionMethod:               storage.CollectionMethod_CORE_BPF,
		AdmissionController:            true,
		AdmissionControllerUpdates:     true,
		AdmissionControllerEvents:      true,
		AdmissionControllerFailOnError: true,
		DynamicConfig: &storage.DynamicClusterConfig{
			DisableAuditLogs: false,
		},
		TolerationsConfig: &storage.TolerationsConfig{
			Disabled: true,
		},
		SlimCollector: true,
		Priority:      10,
		ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		Status: &storage.ClusterStatus{
			SensorVersion: "3.0.0",
		},
		HealthStatus: &storage.ClusterHealthStatus{
			SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		HelmConfig: &storage.CompleteClusterConfig{
			ClusterLabels: map[string]string{"helm": "true"},
		},
		MostRecentSensorId: &storage.SensorDeploymentIdentification{
			SystemNamespaceId: "ns-id",
		},
		AuditLogState: map[string]*storage.AuditLogFileState{
			"node1": {CollectLogsSince: nil},
		},
		InitBundleId:       "init-bundle-123",
		SensorCapabilities: []string{"cap1", "cap2"},
	}

	result := StorageClusterToAPIClusterConfig(cluster)

	assert.Equal(t, cluster.GetId(), result.GetId())
	assert.Equal(t, cluster.GetName(), result.GetName())
	assert.Equal(t, cluster.GetType(), result.GetType())
	assert.Equal(t, cluster.GetLabels(), result.GetLabels())
	assert.Equal(t, cluster.GetMainImage(), result.GetMainImage())
	assert.Equal(t, cluster.GetCollectorImage(), result.GetCollectorImage())
	assert.Equal(t, cluster.GetCentralApiEndpoint(), result.GetCentralApiEndpoint())
	assert.Equal(t, cluster.GetCollectionMethod(), result.GetCollectionMethod())
	assert.Equal(t, cluster.GetAdmissionController(), result.GetAdmissionController())
	assert.Equal(t, cluster.GetAdmissionControllerUpdates(), result.GetAdmissionControllerUpdates())
	assert.Equal(t, cluster.GetAdmissionControllerEvents(), result.GetAdmissionControllerEvents())
	assert.Equal(t, cluster.GetAdmissionControllerFailOnError(), result.GetAdmissionControllerFailOnError())
	protoassert.Equal(t, cluster.GetDynamicConfig(), result.GetDynamicConfig())
	protoassert.Equal(t, cluster.GetTolerationsConfig(), result.GetTolerationsConfig())
	assert.Equal(t, cluster.GetSlimCollector(), result.GetSlimCollector())
	assert.Equal(t, cluster.GetPriority(), result.GetPriority())
	assert.Equal(t, cluster.GetManagedBy(), result.GetManagedBy())
	protoassert.Equal(t, cluster.GetStatus(), result.GetStatus())
	protoassert.Equal(t, cluster.GetHealthStatus(), result.GetHealthStatus())
	protoassert.Equal(t, cluster.GetHelmConfig(), result.GetHelmConfig())
	protoassert.Equal(t, cluster.GetMostRecentSensorId(), result.GetMostRecentSensorId())
	assert.Equal(t, cluster.GetInitBundleId(), result.GetInitBundleId())
	assert.Equal(t, cluster.GetSensorCapabilities(), result.GetSensorCapabilities())
}

func TestStorageClustersToAPIClusterConfigs_Multiple(t *testing.T) {
	clusters := []*storage.Cluster{
		{Id: "id-1", Name: "cluster-1"},
		{Id: "id-2", Name: "cluster-2"},
		{Id: "id-3", Name: "cluster-3"},
	}

	result := StorageClustersToAPIClusterConfigs(clusters)

	assert.Len(t, result, 3)
	for i, c := range clusters {
		assert.Equal(t, c.GetId(), result[i].GetId())
		assert.Equal(t, c.GetName(), result[i].GetName())
	}
}
