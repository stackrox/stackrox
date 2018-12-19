package replicaset

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/retry"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceZeroReplica scales a ReplicaSet down to 0 instances.
func EnforceZeroReplica(client *kubernetes.Clientset, deploymentInfo *central.DeploymentEnforcement) (err error) {
	scaleRequest := &v1beta1.Scale{
		Spec: pkgKubernetes.ScaleToZeroSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentInfo.GetDeploymentName(),
			Namespace: deploymentInfo.GetNamespace(),
		},
	}

	_, err = client.ExtensionsV1beta1().ReplicaSets(deploymentInfo.GetNamespace()).UpdateScale(deploymentInfo.GetDeploymentName(), scaleRequest)
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
