package statefulset

import (
	"context"

	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/retry"
	appsV1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceZeroReplica scales a StatefulSet down to 0 instances.
func EnforceZeroReplica(ctx context.Context, client kubernetes.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	var ss *appsV1.StatefulSet
	ss, err = client.AppsV1beta1().StatefulSets(deploymentInfo.GetNamespace()).Get(ctx, deploymentInfo.GetDeploymentName(), metav1.GetOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}

	err = scaleStatefulSetToZero(ctx, client, ss)
	return retry.MakeRetryable(err)
}

func scaleStatefulSetToZero(ctx context.Context, client kubernetes.Interface, ss *appsV1.StatefulSet) (err error) {
	ss.Spec.Replicas = &[]int32{0}[0]
	_, err = client.AppsV1beta1().StatefulSets(ss.GetNamespace()).Update(ctx, ss, metav1.UpdateOptions{})
	return
}
