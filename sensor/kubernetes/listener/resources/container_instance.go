package resources

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	corev1 "k8s.io/api/core/v1"
)

const (
	dockerContainerIDPrefix = `docker://`
)

func containerInstances(pod *corev1.Pod) []*storage.ContainerInstance {
	podID := getPodID(pod).String()
	result := make([]*storage.ContainerInstance, len(pod.Status.ContainerStatuses))
	for i, c := range pod.Status.ContainerStatuses {
		instID := containerInstanceID(c, pod.Spec.NodeName)
		var ips []string
		if pod.Status.PodIP != "" {
			ips = []string{pod.Status.PodIP}
		}
		result[i] = &storage.ContainerInstance{
			InstanceId:      instID,
			ContainingPodId: podID,
			ContainerIps:    ips,
		}
	}
	return result
}

func containerInstanceID(cs corev1.ContainerStatus, node string) *storage.ContainerInstanceID {
	runtime, runtimeID := parseContainerID(cs.ContainerID)
	return &storage.ContainerInstanceID{
		ContainerRuntime: runtime,
		Id:               runtimeID,
		Node:             node,
	}
}

func parseContainerID(id string) (storage.ContainerRuntime, string) {
	runtime := storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME
	if strings.HasPrefix(id, dockerContainerIDPrefix) {
		runtime = storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME
		id = id[len(dockerContainerIDPrefix):]
	}
	return runtime, id
}
