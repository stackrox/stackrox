package generate

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestClusterConfigToStorage_Nil(t *testing.T) {
	assert.Nil(t, clusterConfigToStorage(nil))
}

func TestClusterConfigToStorage_AllFields(t *testing.T) {
	config := &v1.ClusterConfig{
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
		HelmConfig: &storage.CompleteClusterConfig{
			ClusterLabels: map[string]string{"helm": "true"},
		},
	}

	result := clusterConfigToStorage(config)

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
	protoassert.Equal(t, config.GetHelmConfig(), result.GetHelmConfig())
}
