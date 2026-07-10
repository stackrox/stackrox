package storagetov2

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestAIWorkload(t *testing.T) {
	tests := map[string]struct {
		input    *storage.AIWorkload
		expected *v2.AIWorkload
	}{
		"nil input returns nil": {
			input:    nil,
			expected: nil,
		},
		"converts all fields": {
			input: &storage.AIWorkload{
				Id:             "test-id",
				Namespace:      "test-ns",
				Name:           "test-model",
				ClusterId:      "cluster-1",
				ClusterName:    "my-cluster",
				WorkloadType:   storage.AIWorkload_INFERENCE,
				ModelFormat:    "vLLM",
				StorageUri:     "oci://quay.io/model:latest",
				Runtime:        "vllm-runtime",
				GpuRequests:    "2",
				CpuRequests:    "4",
				MemoryRequests: "40Gi",
				DeploymentMode: "RawDeployment",
				AuthEnabled:    true,
			},
			expected: &v2.AIWorkload{
				Id:             "test-id",
				Namespace:      "test-ns",
				Name:           "test-model",
				ClusterId:      "cluster-1",
				ClusterName:    "my-cluster",
				WorkloadType:   v2.AIWorkload_INFERENCE,
				ModelFormat:    "vLLM",
				StorageUri:     "oci://quay.io/model:latest",
				Runtime:        "vllm-runtime",
				GpuRequests:    "2",
				CpuRequests:    "4",
				MemoryRequests: "40Gi",
				DeploymentMode: "RawDeployment",
				AuthEnabled:    true,
			},
		},
		"converts training workload type": {
			input: &storage.AIWorkload{
				Id:           "training-id",
				WorkloadType: storage.AIWorkload_TRAINING,
			},
			expected: &v2.AIWorkload{
				Id:           "training-id",
				WorkloadType: v2.AIWorkload_TRAINING,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := AIWorkload(tc.input)
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				protoassert.Equal(t, tc.expected, result)
			}
		})
	}
}
