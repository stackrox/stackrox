package pod

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/retry"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	podDeleteOptions = &metaV1.DeleteOptions{
		GracePeriodSeconds: &([]int64{0})[0],
		PropagationPolicy:  &pkgKubernetes.DeletePolicyBackground,
	}
)

// EnforceKill kills the pod holding the container info specified container instance.
func EnforceKill(client *kubernetes.Clientset, containerInfo *central.ContainerInstanceEnforcement) (err error) {
	podID := containerInfo.GetPodId()
	ns := containerInfo.GetDeploymentEnforcement().GetNamespace()

	err = client.CoreV1().Pods(ns).Delete(podID, podDeleteOptions)
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
