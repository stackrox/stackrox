package printer

import (
	"fmt"
	"strconv"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/kubernetes"
)

// GenerateKubeEventViolationMsg constructs violation message for kubernetes event violations.
func GenerateKubeEventViolationMsg(event *storage.KubernetesEvent) *storage.Alert_Violation {
	switch event.GetObject().GetResource() {
	case storage.KubernetesEvent_Object_PODS_EXEC:
		return podExecViolationMsg(event.GetObject().GetName(), event.GetPodExecArgs())
	case storage.KubernetesEvent_Object_PODS_PORTFORWARD:
		return podPortForwardViolationMsg(event.GetObject().GetName(), event.GetPodPortForwardArgs())
	default:
		return defaultViolationMsg(event)
	}
}

func defaultViolationMsg(event *storage.KubernetesEvent) *storage.Alert_Violation {
	return &storage.Alert_Violation{
		Message: fmt.Sprintf("Kubernetes event '%s' detected", kubernetes.EventAsString(event)),
	}
}

func podExecViolationMsg(pod string, args *storage.KubernetesEvent_PodExecArgs) *storage.Alert_Violation {
	return &storage.Alert_Violation{
		Message: fmt.Sprintf("Kubectl exec '%s' into pod '%s' container '%s' detected",
			args.GetCommand(), args.GetContainer(), pod),
		MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
			KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
				Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
					{Key: "pod", Value: pod},
					{Key: "container", Value: args.GetContainer()},
					{Key: "command", Value: args.GetCommand()},
				},
			},
		},
	}
}

func podPortForwardViolationMsg(pod string, args *storage.KubernetesEvent_PodPortForwardArgs) *storage.Alert_Violation {
	port := strconv.Itoa((int)(args.GetPort()))
	return &storage.Alert_Violation{
		Message: fmt.Sprintf("Kubectl port-forward to pod '%s' port '%s' detected", pod, port),
		MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
			KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
				Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
					{Key: "pod", Value: pod},
					{Key: "port", Value: port},
				},
			},
		},
	}
}
