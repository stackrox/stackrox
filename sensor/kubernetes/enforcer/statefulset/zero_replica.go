package statefulset

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	appsV1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceZeroReplica scales a StatefulSet down to 0 instances.
func EnforceZeroReplica(client *kubernetes.Clientset, deploymentInfo *central.DeploymentEnforcement) (err error) {
	var ss *appsV1.StatefulSet
	ss, err = client.AppsV1beta1().StatefulSets(deploymentInfo.GetNamespace()).Get(deploymentInfo.GetDeploymentName(), metav1.GetOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}

	err = scaleStatefulSetToZero(client, ss)
	return retry.MakeRetryable(err)
}

func scaleStatefulSetToZero(client *kubernetes.Clientset, ss *appsV1.StatefulSet) (err error) {
	ss.Spec.Replicas = &[]int32{0}[0]
	_, err = client.AppsV1beta1().StatefulSets(ss.GetNamespace()).Update(ss)
	return
}
