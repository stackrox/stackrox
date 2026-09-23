package dispatcher

import (
	aiworkloadV1 "github.com/stackrox/rox/generated/internalapi/aiworkload/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type InferenceServiceDispatcher struct {
	clusterID string
}

func NewInferenceServiceDispatcher(clusterID string) *InferenceServiceDispatcher {
	return &InferenceServiceDispatcher{clusterID: clusterID}
}

func (d *InferenceServiceDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	if !features.AIWorkloads.Enabled() || !centralcaps.Has(centralsensor.AIWorkloadsSupported) {
		return nil
	}

	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil
	}

	if action == central.ResourceAction_REMOVE_RESOURCE {
		return createEvent(u, &aiworkloadV1.AIWorkload{
			Id:        string(u.GetUID()),
			Name:      u.GetName(),
			Namespace: u.GetNamespace(),
			ClusterId: d.clusterID,
		}, action)
	}

	spec, _ := u.Object["spec"].(map[string]interface{})
	if spec == nil {
		return nil
	}

	predictor, _ := spec["predictor"].(map[string]interface{})
	if predictor == nil {
		return nil
	}

	workload := &aiworkloadV1.AIWorkload{
		Id:           string(u.GetUID()),
		Name:         u.GetName(),
		Namespace:    u.GetNamespace(),
		ClusterId:    d.clusterID,
		WorkloadType: aiworkloadV1.AIWorkload_INFERENCE,
	}

	model, _ := predictor["model"].(map[string]interface{})
	if model != nil {
		workload.ModelFormat = getStringField(model, "modelFormat", "name")
		workload.StorageUri = getStringField(model, "storageUri")
		workload.Runtime = getStringField(model, "runtime")

		workload.CpuRequests = getResourceQuantity(model, []string{"resources", "requests"}, "cpu")
		workload.MemoryRequests = getResourceQuantity(model, []string{"resources", "requests"}, "memory")
		workload.GpuRequests = getResourceQuantity(model, []string{"resources", "requests"}, "nvidia.com/gpu")
	}

	annotations := u.GetAnnotations()
	if annotations != nil {
		workload.DeploymentMode = annotations["serving.kserve.io/deploymentMode"]
		workload.AuthEnabled = annotations["security.opendatahub.io/enable-auth"] == "true"
	}

	return createEvent(u, workload, action)
}
