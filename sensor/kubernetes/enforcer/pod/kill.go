package pod

import (
	roxV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceKill kills the pod holding the container info specified container instance.
func EnforceKill(client *kubernetes.Clientset, containerInfo *roxV1.ContainerInstanceEnforcement) (err error) {
	pID, err := resources.ParsePodID(containerInfo.GetPodId())
	if err != nil {
		return err
	}

	err = client.CoreV1().Pods(pID.Namespace).Delete(pID.Name, &metaV1.DeleteOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
