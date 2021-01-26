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
	var message string
	attrs := []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{{Key: "pod", Value: pod}}

	cmds := stringutils.JoinNonEmpty(", ", args.GetCommands()...)
	if len(cmds) > 0 {
		message = fmt.Sprintf("Kubernetes API received exec '%s' request into pod '%s'", cmds, pod)
	} else {
		message = fmt.Sprintf("Kubernetes API received exec request into pod '%s'", pod)
	}

	if args.GetContainer() != "" {
		message = fmt.Sprintf("%s container '%s'", message, args.GetContainer())
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: "container", Value: args.GetContainer()})
	}

	// Order of attrs-pods, containers and commands
	if len(cmds) > 0 {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: "commands", Value: cmds})
	}

	return &storage.Alert_Violation{
		Message: message,
		MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
			KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
				Attrs: attrs,
			},
		},
	}
}

func podPortForwardViolationMsg(pod string, args *storage.KubernetesEvent_PodPortForwardArgs) *storage.Alert_Violation {
	message := fmt.Sprintf("Kubernetes API received port forward request to pod '%s'", pod)
	attrs := []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{{Key: "pod", Value: pod}}

	if len(args.GetPorts()) > 0 {
		ports := stringutils.JoinInt32(", ", args.GetPorts()...)
		message = fmt.Sprintf("%s ports '%s'", message, ports)
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: "ports", Value: ports})
	}

	return &storage.Alert_Violation{
		Message: message,
		MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
			KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
				Attrs: attrs,
			},
		},
	}
}
