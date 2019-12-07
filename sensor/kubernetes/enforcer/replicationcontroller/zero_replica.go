package replicationcontroller

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceZeroReplica scales a ReplicationController down to 0 instances.
func EnforceZeroReplica(client *kubernetes.Clientset, deploymentInfo *central.DeploymentEnforcement) (err error) {
	scaleRequest := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{Replicas: 0},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentInfo.GetDeploymentName(),
			Namespace: deploymentInfo.GetNamespace(),
		},
	}

	_, err = client.CoreV1().ReplicationControllers(deploymentInfo.GetNamespace()).UpdateScale(deploymentInfo.GetDeploymentName(), scaleRequest)
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
