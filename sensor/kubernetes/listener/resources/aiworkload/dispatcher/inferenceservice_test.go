package dispatcher

import (
	"testing"

	aiworkloadV1 "github.com/stackrox/rox/generated/internalapi/aiworkload/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestInferenceServiceDispatcher_ProcessEvent(t *testing.T) {
	t.Setenv(features.AIWorkloads.EnvVar(), "true")
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.AIWorkloadsSupported})
	t.Cleanup(func() { centralcaps.Set(nil) })

	dispatcher := NewInferenceServiceDispatcher("test-cluster-id")

	tests := map[string]struct {
		obj            *unstructured.Unstructured
		action         central.ResourceAction
		expectedNil    bool
		expectedFields func(t *testing.T, workload *aiworkloadV1.AIWorkload)
	}{
		"extracts all fields from a fully populated InferenceService": {
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"uid":       "test-uid",
						"name":      "granite-model",
						"namespace": "ai-project",
						"annotations": map[string]interface{}{
							"serving.kserve.io/deploymentMode":    "RawDeployment",
							"security.opendatahub.io/enable-auth": "true",
						},
					},
					"spec": map[string]interface{}{
						"predictor": map[string]interface{}{
							"model": map[string]interface{}{
								"modelFormat": map[string]interface{}{
									"name": "vLLM",
								},
								"storageUri": "oci://quay.io/modh/granite-8b:latest",
								"runtime":    "vllm-runtime",
								"resources": map[string]interface{}{
									"requests": map[string]interface{}{
										"cpu":            "4",
										"memory":         "40Gi",
										"nvidia.com/gpu": "2",
									},
								},
							},
						},
					},
				},
			},
			action: central.ResourceAction_CREATE_RESOURCE,
			expectedFields: func(t *testing.T, workload *aiworkloadV1.AIWorkload) {
				assert.Equal(t, "test-uid", workload.GetId())
				assert.Equal(t, "granite-model", workload.GetName())
				assert.Equal(t, "ai-project", workload.GetNamespace())
				assert.Equal(t, "test-cluster-id", workload.GetClusterId())
				assert.Equal(t, aiworkloadV1.AIWorkload_INFERENCE, workload.GetWorkloadType())
				assert.Equal(t, "vLLM", workload.GetModelFormat())
				assert.Equal(t, "oci://quay.io/modh/granite-8b:latest", workload.GetStorageUri())
				assert.Equal(t, "vllm-runtime", workload.GetRuntime())
				assert.Equal(t, "4", workload.GetCpuRequests())
				assert.Equal(t, "40Gi", workload.GetMemoryRequests())
				assert.Equal(t, "2", workload.GetGpuRequests())
				assert.Equal(t, "RawDeployment", workload.GetDeploymentMode())
				assert.True(t, workload.GetAuthEnabled())
			},
		},
		"handles missing model section": {
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"uid":       "test-uid-2",
						"name":      "no-model",
						"namespace": "ai-project",
					},
					"spec": map[string]interface{}{
						"predictor": map[string]interface{}{},
					},
				},
			},
			action: central.ResourceAction_CREATE_RESOURCE,
			expectedFields: func(t *testing.T, workload *aiworkloadV1.AIWorkload) {
				assert.Equal(t, "no-model", workload.GetName())
				assert.Equal(t, aiworkloadV1.AIWorkload_INFERENCE, workload.GetWorkloadType())
				assert.Empty(t, workload.GetModelFormat())
				assert.Empty(t, workload.GetStorageUri())
			},
		},
		"handles remove action with minimal fields": {
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"uid":       "test-uid-3",
						"name":      "deleted-model",
						"namespace": "ai-project",
					},
				},
			},
			action: central.ResourceAction_REMOVE_RESOURCE,
			expectedFields: func(t *testing.T, workload *aiworkloadV1.AIWorkload) {
				assert.Equal(t, "test-uid-3", workload.GetId())
				assert.Equal(t, "deleted-model", workload.GetName())
				assert.Equal(t, "ai-project", workload.GetNamespace())
			},
		},
		"returns nil when spec is missing": {
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"uid":  "test-uid-4",
						"name": "bad-resource",
					},
				},
			},
			action:      central.ResourceAction_CREATE_RESOURCE,
			expectedNil: true,
		},
		"returns nil when predictor is missing": {
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"uid":  "test-uid-5",
						"name": "no-predictor",
					},
					"spec": map[string]interface{}{},
				},
			},
			action:      central.ResourceAction_CREATE_RESOURCE,
			expectedNil: true,
		},
		"auth disabled when annotation missing": {
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"uid":       "test-uid-6",
						"name":      "no-auth",
						"namespace": "ai-project",
					},
					"spec": map[string]interface{}{
						"predictor": map[string]interface{}{
							"model": map[string]interface{}{
								"modelFormat": map[string]interface{}{
									"name": "pytorch",
								},
							},
						},
					},
				},
			},
			action: central.ResourceAction_CREATE_RESOURCE,
			expectedFields: func(t *testing.T, workload *aiworkloadV1.AIWorkload) {
				assert.False(t, workload.GetAuthEnabled())
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := dispatcher.ProcessEvent(tc.obj, nil, tc.action)
			if tc.expectedNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			require.Len(t, result.ForwardMessages, 1)
			event := result.ForwardMessages[0]
			workload := event.GetAiWorkload()
			require.NotNil(t, workload)
			tc.expectedFields(t, workload)
		})
	}
}
