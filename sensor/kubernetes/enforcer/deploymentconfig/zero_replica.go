package deploymentconfig

import (
	"context"
	"encoding/json"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

var deploymentConfigGVR = schema.GroupVersionResource{
	Group:    "apps.openshift.io",
	Version:  "v1",
	Resource: "deploymentconfigs",
}

// EnforceZeroReplica scales a DeploymentConfig down to 0 instances
// using the dynamic client to avoid importing the typed OpenShift
// apps client (which registers scheme types at init, adding ~10 MB RSS).
func EnforceZeroReplica(ctx context.Context, client dynamic.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	patch, _ := json.Marshal(map[string]any{
		"spec": map[string]any{
			"replicas": 0,
		},
	})

	_, err = client.Resource(deploymentConfigGVR).
		Namespace(deploymentInfo.GetNamespace()).
		Patch(ctx, deploymentInfo.GetDeploymentName(), k8sTypes.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
