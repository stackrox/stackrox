package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestConvertStorageClusterToAPI_Nil(t *testing.T) {
	assert.Nil(t, convertStorageClusterToAPI(nil))
}

func TestConvertAPIClusterToStorage_Nil(t *testing.T) {
	assert.Nil(t, convertAPIClusterToStorage(nil))
}

func TestConvertStorageClustersToAPI_Nil(t *testing.T) {
	assert.Nil(t, convertStorageClustersToAPI(nil))
}

func TestConvertStorageClustersToAPI_Empty(t *testing.T) {
	result := convertStorageClustersToAPI([]*storage.Cluster{})
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestConvertStorageClusterToAPI_AllFields(t *testing.T) {
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

	result := convertStorageClusterToAPI(cluster)

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

func TestConvertAPIClusterToStorage_UserSettableFields(t *testing.T) {
	config := &v1.ClusterConfig{
		Id:                             "test-id",
		Name:                           "test-cluster",
		Type:                           storage.ClusterType_KUBERNETES_CLUSTER,
		Labels:                         map[string]string{"env": "dev"},
		MainImage:                      "quay.io/stackrox/main:latest",
		CollectorImage:                 "quay.io/stackrox/collector:latest",
		CentralApiEndpoint:             "central.stackrox:443",
		CollectionMethod:               storage.CollectionMethod_CORE_BPF,
		AdmissionController:            true,
		AdmissionControllerUpdates:     true,
		AdmissionControllerEvents:      true,
		AdmissionControllerFailOnError: true,
		DynamicConfig: &storage.DynamicClusterConfig{
			DisableAuditLogs: true,
		},
		TolerationsConfig: &storage.TolerationsConfig{
			Disabled: true,
		},
		SlimCollector: true,
		Priority:      5,
		ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
	}

	result := convertAPIClusterToStorage(config)

	assert.Equal(t, config.GetId(), result.GetId())
	assert.Equal(t, config.GetName(), result.GetName())
	assert.Equal(t, config.GetType(), result.GetType())
	assert.Equal(t, config.GetLabels(), result.GetLabels())
	assert.Equal(t, config.GetMainImage(), result.GetMainImage())
	assert.Equal(t, config.GetCollectorImage(), result.GetCollectorImage())
	assert.Equal(t, config.GetCentralApiEndpoint(), result.GetCentralApiEndpoint())
	assert.Equal(t, config.GetCollectionMethod(), result.GetCollectionMethod())
	assert.Equal(t, config.GetAdmissionController(), result.GetAdmissionController())
	assert.Equal(t, config.GetAdmissionControllerUpdates(), result.GetAdmissionControllerUpdates())
	assert.Equal(t, config.GetAdmissionControllerEvents(), result.GetAdmissionControllerEvents())
	assert.Equal(t, config.GetAdmissionControllerFailOnError(), result.GetAdmissionControllerFailOnError())
	protoassert.Equal(t, config.GetDynamicConfig(), result.GetDynamicConfig())
	protoassert.Equal(t, config.GetTolerationsConfig(), result.GetTolerationsConfig())
	assert.Equal(t, config.GetSlimCollector(), result.GetSlimCollector())
	assert.Equal(t, config.GetPriority(), result.GetPriority())
	assert.Equal(t, config.GetManagedBy(), result.GetManagedBy())
}

func TestConvertAPIClusterToStorage_ServerManagedFieldsIgnored(t *testing.T) {
	config := &v1.ClusterConfig{
		Id:   "test-id",
		Name: "test-cluster",
		// Server-managed fields that should be silently discarded
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

	result := convertAPIClusterToStorage(config)

	assert.Equal(t, "test-id", result.GetId())
	assert.Equal(t, "test-cluster", result.GetName())
	assert.Nil(t, result.GetStatus())
	assert.Nil(t, result.GetHealthStatus())
	assert.Nil(t, result.GetHelmConfig())
	assert.Nil(t, result.GetMostRecentSensorId())
	assert.Nil(t, result.GetAuditLogState())
	assert.Empty(t, result.GetInitBundleId())
	assert.Nil(t, result.GetSensorCapabilities())
}

func TestConvertStorageClustersToAPI_Multiple(t *testing.T) {
	clusters := []*storage.Cluster{
		{Id: "id-1", Name: "cluster-1"},
		{Id: "id-2", Name: "cluster-2"},
		{Id: "id-3", Name: "cluster-3"},
	}

	result := convertStorageClustersToAPI(clusters)

	assert.Len(t, result, 3)
	for i, c := range clusters {
		assert.Equal(t, c.GetId(), result[i].GetId())
		assert.Equal(t, c.GetName(), result[i].GetName())
	}
}
