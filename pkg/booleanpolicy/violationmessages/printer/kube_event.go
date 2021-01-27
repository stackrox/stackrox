package printer

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/stringutils"
)

// GenerateKubeEventViolationMsg constructs violation message for kubernetes event violations.
func GenerateKubeEventViolationMsg(event *storage.KubernetesEvent) *storage.Alert_Violation {
	var message string
	var attrs []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr

	switch event.GetObject().GetResource() {
	case storage.KubernetesEvent_Object_PODS_EXEC:
		message, attrs = podExecViolationMsg(event.GetObject().GetName(), event.GetPodExecArgs())
	case storage.KubernetesEvent_Object_PODS_PORTFORWARD:
		message, attrs = podPortForwardViolationMsg(event.GetObject().GetName(), event.GetPodPortForwardArgs())
	default:
		message, attrs = defaultViolationMsg(event)
	}

	return &storage.Alert_Violation{
		Message: message,
		MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
			KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
				Attrs: attrs,
			},
		},
		Type: storage.Alert_Violation_K8S_EVENT,
		Time: event.GetTimestamp(),
	}
}

func defaultViolationMsg(event *storage.KubernetesEvent) (string, []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr) {
	return fmt.Sprintf("Kubernetes API request '%s' detected", kubernetes.EventAsString(event)), nil
}

func podExecViolationMsg(pod string, args *storage.KubernetesEvent_PodExecArgs) (string, []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr) {
	cmds := stringutils.JoinNonEmpty(" ", args.GetCommands()...)
	return getExecMsgHeader(pod, args.GetContainer(), cmds), getExecMsgViolationAttr(pod, args.GetContainer(), cmds)
}

func podPortForwardViolationMsg(pod string, args *storage.KubernetesEvent_PodPortForwardArgs) (string, []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr) {
	ports := stringutils.JoinInt32(", ", args.GetPorts()...)
	return getPFMsgHeader(pod, ports), getPFMsgViolationAttr(pod, ports)
}

func getExecMsgHeader(pod, container, cmds string) string {
	prefix := "Kubernetes API received exec"
	if pod != "" {
		pod = fmt.Sprintf("into pod '%s'", pod)
	}

	if container != "" {
		container = fmt.Sprintf("container '%s'", container)
	}

	if cmds != "" {
		cmds = fmt.Sprintf("'%s'", cmds)
	}
	return stringutils.JoinNonEmpty(" ", prefix, cmds, "request", pod, container)
}

func getExecMsgViolationAttr(pod, container, cmds string) []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr {
	attrs := make([]*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr, 0, 3)
	if pod != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: "pod", Value: pod})
	}

	if container != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: "container", Value: container})
	}

	if cmds != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: "commands", Value: cmds})
	}
	return attrs
}

func getPFMsgHeader(pod, ports string) string {
	prefix := "Kubernetes API received port forward request"
	if pod == "" {
		return prefix
	}

	pod = fmt.Sprintf("to pod '%s'", pod)

	if ports != "" {
		ports = fmt.Sprintf("ports '%s'", ports)
	}
	return stringutils.JoinNonEmpty(" ", prefix, pod, ports)
}

func getPFMsgViolationAttr(pod, ports string) []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr {
	attrs := make([]*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr, 0, 2)
	if pod != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: "pod", Value: pod})
	}

	if ports != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: "ports", Value: ports})
	}
	return attrs
}
