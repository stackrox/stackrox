package printer

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/stringutils"
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
		Message: fmt.Sprintf("Kubernetes API request '%s' detected", kubernetes.EventAsString(event)),
	}
}

func podExecViolationMsg(pod string, args *storage.KubernetesEvent_PodExecArgs) *storage.Alert_Violation {
	cmds := stringutils.JoinNonEmpty(", ", args.GetCommands()...)
	return &storage.Alert_Violation{
		Message: fmt.Sprintf("Kubernetes API received exec '%s' request into pod '%s' container '%s'",
			cmds, pod, args.GetContainer()),
		MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
			KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
				Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
					{Key: "pod", Value: pod},
					{Key: "container", Value: args.GetContainer()},
					{Key: "commands", Value: cmds},
				},
			},
		},
	}
}

func podPortForwardViolationMsg(pod string, args *storage.KubernetesEvent_PodPortForwardArgs) *storage.Alert_Violation {
	ports := stringutils.JoinInt32(", ", args.GetPorts()...)
	return &storage.Alert_Violation{
		Message: fmt.Sprintf("Kubernetes API received port forward request to pod '%s' ports '%s'", pod, ports),
		MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
			KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
				Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
					{Key: "pod", Value: pod},
					{Key: "ports", Value: ports},
				},
			},
		},
	}
}
