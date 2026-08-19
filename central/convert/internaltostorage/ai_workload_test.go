package internaltostorage

import (
	"testing"

	aiworkloadV1 "github.com/stackrox/rox/generated/internalapi/aiworkload/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestAIWorkload(t *testing.T) {
	tests := map[string]struct {
		input    *aiworkloadV1.AIWorkload
		expected *storage.AIWorkload
	}{
		"nil input returns nil": {
			input:    nil,
			expected: nil,
		},
		"converts all fields": {
			input: &aiworkloadV1.AIWorkload{
				Id:             "test-id",
				Namespace:      "test-ns",
				Name:           "test-model",
				ClusterId:      "cluster-1",
				WorkloadType:   aiworkloadV1.AIWorkload_INFERENCE,
				ModelFormat:    "vLLM",
				StorageUri:     "oci://quay.io/model:latest",
				Runtime:        "vllm-runtime",
				GpuRequests:    "2",
				CpuRequests:    "4",
				MemoryRequests: "40Gi",
				DeploymentMode: "RawDeployment",
				AuthEnabled:    true,
			},
			expected: &storage.AIWorkload{
				Id:             "test-id",
				Namespace:      "test-ns",
				Name:           "test-model",
				ClusterId:      "cluster-1",
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
		},
		"converts training workload type": {
			input: &aiworkloadV1.AIWorkload{
				Id:           "training-id",
				WorkloadType: aiworkloadV1.AIWorkload_TRAINING,
			},
			expected: &storage.AIWorkload{
				Id:           "training-id",
				WorkloadType: storage.AIWorkload_TRAINING,
			},
		},
		"converts unknown workload type": {
			input: &aiworkloadV1.AIWorkload{
				Id:           "unknown-id",
				WorkloadType: aiworkloadV1.AIWorkload_UNKNOWN,
			},
			expected: &storage.AIWorkload{
				Id:           "unknown-id",
				WorkloadType: storage.AIWorkload_UNKNOWN,
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
