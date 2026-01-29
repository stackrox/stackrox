package printer

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	// PodKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote a pod.
	PodKey = "pod"
	// ContainerKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote a container.
	ContainerKey = "container"
	// APIVerbKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote the kubernetes API verb.
	APIVerbKey = "Verb"
	// UsernameKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote the name of the user taking the action.
	UsernameKey = "Username"
	// UserGroupsKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote the groups of the user taking the action.
	UserGroupsKey = "Groups"
	// ImpersonatedUsernameKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote the name of the impersonated user taking the action.
	ImpersonatedUsernameKey = "Impersonated Username"
	// ImpersonatedUserGroupsKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote the groups of the impersonated user taking the action.
	ImpersonatedUserGroupsKey = "Impersonated Groups"
	// ResourceURIKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote the resource URI.
	ResourceURIKey = "Resource"
	// UserAgentKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote the user agent.
	UserAgentKey = "User Agent"
	// IPAddressKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote the IP Address.
	IPAddressKey = "IP address"
	// PortsKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote the (port-forward) ports.
	PortsKey = "ports"
	// CommandsKey is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote the (pod exec) commands.
	CommandsKey = "commands"
)

type attributeOptions struct {
	skipVerb        bool
	skipResourceURI bool
}

// GenerateKubeEventViolationMsg constructs violation message for kubernetes event violations.
func GenerateKubeEventViolationMsg(event *storage.KubernetesEvent) *storage.Alert_Violation {
	var message string
	var attrs []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr

	switch event.GetObject().GetResource() {
	case storage.KubernetesEvent_Object_PODS_EXEC:
		message, attrs = podExecViolationMsg(event)
	case storage.KubernetesEvent_Object_PODS_PORTFORWARD:
		message, attrs = podPortForwardViolationMsg(event)
	case storage.KubernetesEvent_Object_PODS_ATTACH:
		message, attrs = podAttachViolationMsg(event)
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
	return getDefaultViolationMsgHeader(event), getDefaultViolationMsgViolationAttr(event, &attributeOptions{})
}

func podExecViolationMsg(event *storage.KubernetesEvent) (string, []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr) {
	return getExecMsgHeader(event), getExecMsgViolationAttr(event)
}

func podPortForwardViolationMsg(event *storage.KubernetesEvent) (string, []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr) {
	return getPFMsgHeader(event), getPFMsgViolationAttr(event)
}

func podAttachViolationMsg(event *storage.KubernetesEvent) (string, []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr) {
	return getAttachMsgHeader(event), getAttachMsgViolationAttr(event)
}

func getDefaultViolationMsgHeader(event *storage.KubernetesEvent) string {
	object := event.GetObject()
	readableResourceName := strings.ToLower(object.GetResource().String())

	var singularResourceName string
	if strings.HasSuffix(readableResourceName, "ies") {
		singularResourceName = strings.TrimSuffix(readableResourceName, "ies")
		singularResourceName = fmt.Sprintf("%sy", singularResourceName)
	} else {
		singularResourceName = strings.TrimSuffix(readableResourceName, "s")
	}
	singularResourceName = strings.ReplaceAll(singularResourceName, "_", " ")
	readableResourceName = strings.ReplaceAll(readableResourceName, "_", " ")

	var header string
	if object.GetName() == "" {
		header = fmt.Sprintf("Access to %s", readableResourceName)
		if object.GetNamespace() != "" {
			header = fmt.Sprintf("%s in namespace \"%s\"", header, object.GetNamespace())

		}
		return header
	}

	header = fmt.Sprintf("Access to %s \"%s\"", singularResourceName, object.GetName())
	if object.GetNamespace() != "" {
		header = fmt.Sprintf("%s in namespace \"%s\"", header, object.GetNamespace())

	}
	return header
}

func getDefaultViolationMsgViolationAttr(event *storage.KubernetesEvent, options *attributeOptions) []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr {
	attrs := make([]*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr, 0, 8)

	// the proto guarantees that this will always have a value (even if it's UNKNOWN)
	if !options.skipVerb {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: APIVerbKey, Value: event.GetApiVerb().String()})
	}

	if event.GetUser() != nil {
		if event.GetUser().GetUsername() != "" {
			attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: UsernameKey, Value: event.GetUser().GetUsername()})
		}
		if len(event.GetUser().GetGroups()) > 0 {
			attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: UserGroupsKey, Value: strings.Join(event.GetUser().GetGroups(), ", ")})
		}
	}

	if event.GetUserAgent() != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: UserAgentKey, Value: event.GetUserAgent()})
	}

	if len(event.GetSourceIps()) > 0 {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: IPAddressKey, Value: strings.Join(event.GetSourceIps(), ", ")})
	}

	if !options.skipResourceURI {
		if uriParts := strings.Split(event.GetRequestUri(), "?"); len(uriParts) > 0 && !stringutils.AllEmpty(uriParts...) {
			attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: ResourceURIKey, Value: uriParts[0]})
		}
	}

	if event.GetImpersonatedUser() != nil {
		if event.GetImpersonatedUser().GetUsername() != "" {
			attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: ImpersonatedUsernameKey, Value: event.GetImpersonatedUser().GetUsername()})
		}
		if len(event.GetImpersonatedUser().GetGroups()) > 0 {
			attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: ImpersonatedUserGroupsKey, Value: strings.Join(event.GetImpersonatedUser().GetGroups(), ", ")})
		}
	}
	return attrs
}

func getExecMsgHeader(event *storage.KubernetesEvent) string {
	pod := event.GetObject().GetName()
	container := event.GetPodExecArgs().GetContainer()
	cmds := stringutils.JoinNonEmpty(" ", event.GetPodExecArgs().GetCommands()...)

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

// podEventAttributes holds optional event-specific attributes for pod-related events.
type podEventAttributes struct {
	container string
	commands  string
	ports     string
}

// getPodEventViolationAttr builds violation attributes for pod events (exec, attach, port-forward).
// It handles the common pod attribute and appends event-specific and default attributes.
func getPodEventViolationAttr(event *storage.KubernetesEvent, eventSpecificAttr podEventAttributes) []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr {
	attrs := make([]*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr, 0, 4)

	// Common: pod name
	if pod := event.GetObject().GetName(); pod != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: PodKey, Value: pod})
	}

	// Event-specific attributes
	if eventSpecificAttr.container != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: ContainerKey, Value: eventSpecificAttr.container})
	}
	if eventSpecificAttr.commands != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: CommandsKey, Value: eventSpecificAttr.commands})
	}
	if eventSpecificAttr.ports != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: PortsKey, Value: eventSpecificAttr.ports})
	}

	// Common: default attrs
	attrs = append(attrs, getDefaultViolationMsgViolationAttr(event, &attributeOptions{skipVerb: true, skipResourceURI: true})...)
	return attrs
}

func getExecMsgViolationAttr(event *storage.KubernetesEvent) []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr {
	args := event.GetPodExecArgs()
	return getPodEventViolationAttr(event, podEventAttributes{
		container: args.GetContainer(),
		commands:  stringutils.JoinNonEmpty(" ", args.GetCommands()...),
	})
}

func getPFMsgHeader(event *storage.KubernetesEvent) string {
	pod := event.GetObject().GetName()
	ports := stringutils.JoinInt32(", ", event.GetPodPortForwardArgs().GetPorts()...)

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

func getPFMsgViolationAttr(event *storage.KubernetesEvent) []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr {
	return getPodEventViolationAttr(event, podEventAttributes{
		ports: stringutils.JoinInt32(", ", event.GetPodPortForwardArgs().GetPorts()...),
	})
}

func getAttachMsgHeader(event *storage.KubernetesEvent) string {
	pod := event.GetObject().GetName()
	container := event.GetPodAttachArgs().GetContainer()

	prefix := "Kubernetes API received attach"
	if pod != "" {
		pod = fmt.Sprintf("to pod '%s'", pod)
	}

	if container != "" {
		container = fmt.Sprintf("container '%s'", container)
	}

	return stringutils.JoinNonEmpty(" ", prefix, "request", pod, container)
}

func getAttachMsgViolationAttr(event *storage.KubernetesEvent) []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr {
	return getPodEventViolationAttr(event, podEventAttributes{
		container: event.GetPodAttachArgs().GetContainer(),
	})
}
