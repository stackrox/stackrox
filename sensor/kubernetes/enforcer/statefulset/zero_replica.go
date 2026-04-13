package statefulset

import (
	"context"
	"encoding/json"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// EnforceZeroReplica scales a StatefulSet down to 0 instances.
func EnforceZeroReplica(ctx context.Context, dynClient dynamic.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	patch, _ := json.Marshal(map[string]any{
		"spec": map[string]any{
			"replicas": 0,
		},
	})

	_, err = dynClient.Resource(client.StatefulSetGVR).
		Namespace(deploymentInfo.GetNamespace()).
		Patch(ctx, deploymentInfo.GetDeploymentName(), k8sTypes.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
