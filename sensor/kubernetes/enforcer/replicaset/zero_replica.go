package replicaset

import (
	"context"

	"github.com/stackrox/stackrox/generated/internalapi/central"
	pkgKubernetes "github.com/stackrox/stackrox/pkg/kubernetes"
	"github.com/stackrox/stackrox/pkg/retry"
	autoscalingV1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceZeroReplica scales a ReplicaSet down to 0 instances.
func EnforceZeroReplica(ctx context.Context, client kubernetes.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	scaleRequest := &autoscalingV1.Scale{
		Spec: pkgKubernetes.ScaleToZeroSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentInfo.GetDeploymentName(),
			Namespace: deploymentInfo.GetNamespace(),
		},
	}

	_, err = client.AppsV1().ReplicaSets(deploymentInfo.GetNamespace()).UpdateScale(ctx, deploymentInfo.GetDeploymentName(), scaleRequest, metav1.UpdateOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
