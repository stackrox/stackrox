package pod

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/retry"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	podDeleteOptions = metaV1.DeleteOptions{
		GracePeriodSeconds: &([]int64{0})[0],
		PropagationPolicy:  &pkgKubernetes.DeletePolicyBackground,
	}
)

// EnforceKill kills the pod holding the container info specified container instance.
func EnforceKill(ctx context.Context, client kubernetes.Interface, containerInfo *central.ContainerInstanceEnforcement) (bool, error) {
	podID := containerInfo.GetPodId()
	ns := containerInfo.GetDeploymentEnforcement().GetNamespace()

	err := client.CoreV1().Pods(ns).Delete(ctx, podID, podDeleteOptions)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return true, retry.MakeRetryable(err)
	}
	return true, nil
}
