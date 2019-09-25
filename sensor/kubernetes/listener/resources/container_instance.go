package resources

import (
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	corev1 "k8s.io/api/core/v1"
)

var (
	knownRuntimes = map[string]storage.ContainerRuntime{
		"docker": storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
		"cri-o":  storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME,
	}
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

		if c.State.Running != nil {
			startTime, err := types.TimestampProto(c.State.Running.StartedAt.Time)
			if err != nil {
				log.Error(err)
			}
			result[i].Started = startTime
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
	parts := strings.SplitN(id, "://", 2)
	if len(parts) == 2 {
		id = parts[1]
		if rt, ok := knownRuntimes[parts[0]]; ok {
			runtime = rt
		}
	}
	return runtime, id
}
