package internaltostorage

import (
	aiworkloadV1 "github.com/stackrox/rox/generated/internalapi/aiworkload/v1"
	"github.com/stackrox/rox/generated/storage"
)

func AIWorkload(w *aiworkloadV1.AIWorkload) *storage.AIWorkload {
	if w == nil {
		return nil
	}
	return &storage.AIWorkload{
		Id:             w.GetId(),
		Namespace:      w.GetNamespace(),
		Name:           w.GetName(),
		ClusterId:      w.GetClusterId(),
		WorkloadType:   convertAIWorkloadType(w.GetWorkloadType()),
		ModelFormat:    w.GetModelFormat(),
		StorageUri:     w.GetStorageUri(),
		Runtime:        w.GetRuntime(),
		GpuRequests:    w.GetGpuRequests(),
		CpuRequests:    w.GetCpuRequests(),
		MemoryRequests: w.GetMemoryRequests(),
		DeploymentMode: w.GetDeploymentMode(),
		AuthEnabled:    w.GetAuthEnabled(),
	}
}

func convertAIWorkloadType(t aiworkloadV1.AIWorkload_WorkloadType) storage.AIWorkload_WorkloadType {
	switch t {
	case aiworkloadV1.AIWorkload_INFERENCE:
		return storage.AIWorkload_INFERENCE
	case aiworkloadV1.AIWorkload_TRAINING:
		return storage.AIWorkload_TRAINING
	default:
		return storage.AIWorkload_UNKNOWN
	}
}
