package pod

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceKill kills the pod holding the container info specified container instance.
func EnforceKill(client *kubernetes.Clientset, containerInfo *central.ContainerInstanceEnforcement) (err error) {
	podID := containerInfo.GetPodId()
	ns := containerInfo.GetDeploymentEnforcement().GetNamespace()

	var gracePeriod int64
	err = client.CoreV1().Pods(ns).Delete(podID, &metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
