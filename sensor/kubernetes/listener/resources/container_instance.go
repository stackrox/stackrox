package resources

import (
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	dockerContainerIDPrefix = `docker://`
)

func containerInstances(pod corev1.Pod) []*v1.ContainerInstance {
	podID := getPodID(pod).String()
	result := make([]*v1.ContainerInstance, len(pod.Status.ContainerStatuses))
	for i, c := range pod.Status.ContainerStatuses {
		instID := containerInstanceID(c, pod.Spec.NodeName)
		var ips []string
		if pod.Status.PodIP != "" {
			ips = []string{pod.Status.PodIP}
		}
		result[i] = &v1.ContainerInstance{
			InstanceId:      instID,
			ContainingPodId: podID,
			ContainerIps:    ips,
		}
	}
	return result
}

func containerInstanceID(cs corev1.ContainerStatus, node string) *v1.ContainerInstanceID {
	runtime, runtimeID := parseContainerID(cs.ContainerID)
	return &v1.ContainerInstanceID{
		ContainerRuntime: runtime,
		Id:               runtimeID,
		Node:             node,
	}
}

func parseContainerID(id string) (v1.ContainerRuntime, string) {
	runtime := v1.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME
	if strings.HasPrefix(id, dockerContainerIDPrefix) {
		runtime = v1.ContainerRuntime_DOCKER_CONTAINER_RUNTIME
		id = id[len(dockerContainerIDPrefix):]
	}
	return runtime, id
}
