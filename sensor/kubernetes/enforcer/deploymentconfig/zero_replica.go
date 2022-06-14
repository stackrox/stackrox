package deploymentconfig

import (
	"context"

	"github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/retry"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnforceZeroReplica scales a deployment down to 0 instances.
func EnforceZeroReplica(ctx context.Context, client versioned.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	scaleRequest := &v1beta1.Scale{
		Spec: v1beta1.ScaleSpec{
			Replicas: 0,
		},
	}

	_, err = client.AppsV1().DeploymentConfigs(deploymentInfo.GetNamespace()).UpdateScale(ctx, deploymentInfo.GetDeploymentName(), scaleRequest, metav1.UpdateOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
