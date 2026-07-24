package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func AIWorkload(w *storage.AIWorkload) *v2.AIWorkload {
	if w == nil {
		return nil
	}
	return &v2.AIWorkload{
		Id:             w.GetId(),
		Namespace:      w.GetNamespace(),
		Name:           w.GetName(),
		ClusterId:      w.GetClusterId(),
		ClusterName:    w.GetClusterName(),
		WorkloadType:   convertAIWorkloadType(w.GetWorkloadType()),
		ModelFormat:    w.GetModelFormat(),
		StorageUri:     w.GetStorageUri(),
		Runtime:        w.GetRuntime(),
		GpuRequests:    w.GetGpuRequests(),
		CpuRequests:    w.GetCpuRequests(),
		MemoryRequests: w.GetMemoryRequests(),
		DeploymentMode: w.GetDeploymentMode(),
		AuthEnabled:    w.GetAuthEnabled(),
		LastUpdated:    w.GetLastUpdated(),
	}
}

func convertAIWorkloadType(t storage.AIWorkload_WorkloadType) v2.AIWorkload_WorkloadType {
	switch t {
	case storage.AIWorkload_INFERENCE:
		return v2.AIWorkload_INFERENCE
	case storage.AIWorkload_TRAINING:
		return v2.AIWorkload_TRAINING
	default:
		return v2.AIWorkload_UNKNOWN
	}
}
