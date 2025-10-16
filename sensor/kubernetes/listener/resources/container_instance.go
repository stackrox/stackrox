package resources

import (
	"github.com/stackrox/rox/generated/storage"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/k8sutil"
	podUtils "github.com/stackrox/rox/pkg/pods/utils"
	"github.com/stackrox/rox/pkg/protocompat"
	corev1 "k8s.io/api/core/v1"
)

func containerInstances(pod *corev1.Pod) []*storage.ContainerInstance {
	podID := podUtils.GetPodIDFromV1Pod(pod).String()
	result := make([]*storage.ContainerInstance, len(pod.Status.ContainerStatuses))
	for i, c := range pod.Status.ContainerStatuses {
		instID := containerInstanceID(c, pod.Spec.NodeName)
		var ips []string
		if pod.Status.PodIP != "" {
			ips = []string{pod.Status.PodIP}
		}
		ci := &storage.ContainerInstance{}
		ci.SetInstanceId(instID)
		ci.SetContainingPodId(podID)
		ci.SetContainerIps(ips)
		result[i] = ci

		// Note: Only one of Running/Terminated/Waiting will be set.
		if c.State.Running != nil {
			startTime, err := protocompat.ConvertTimeToTimestampOrError(c.State.Running.StartedAt.Time)
			if err != nil {
				log.Errorf("converting start time from Kubernetes (%v) to proto: %v", c.State.Running.StartedAt.Time, err)
			}
			result[i].SetStarted(startTime)
		}

		result[i].SetContainerName(c.Name)

		// Track terminated containers.
		if terminated := c.State.Terminated; terminated != nil {
			startTime, err := protocompat.ConvertTimeToTimestampOrError(terminated.StartedAt.Time)
			if err != nil {
				log.Errorf("converting start time from Kubernetes (%v) to proto: %v", terminated.StartedAt.Time, err)
			}
			endTime, err := protocompat.ConvertTimeToTimestampOrError(terminated.FinishedAt.Time)
			if err != nil {
				log.Errorf("converting finish time from Kubernetes (%v) to proto: %v", terminated.FinishedAt.Time, err)
			}
			result[i].SetStarted(startTime)
			result[i].SetFinished(endTime)
			result[i].SetExitCode(terminated.ExitCode)
			result[i].SetTerminationReason(terminated.Reason)
		}
		if digest := imageUtils.ExtractImageDigest(c.ImageID); digest != "" {
			result[i].SetImageDigest(digest)
		}
	}
	return result
}

func containerInstanceID(cs corev1.ContainerStatus, node string) *storage.ContainerInstanceID {
	runtime, runtimeID := parseContainerID(cs.ContainerID)
	ciid := &storage.ContainerInstanceID{}
	ciid.SetContainerRuntime(runtime)
	ciid.SetId(runtimeID)
	ciid.SetNode(node)
	return ciid
}

func parseContainerID(id string) (storage.ContainerRuntime, string) {
	return k8sutil.ParseContainerRuntimeString(id)
}
