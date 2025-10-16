package printer

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
	"google.golang.org/protobuf/proto"
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
	default:
		message, attrs = defaultViolationMsg(event)
	}

	avk := &storage.Alert_Violation_KeyValueAttrs{}
	avk.SetAttrs(attrs)
	av := &storage.Alert_Violation{}
	av.SetMessage(message)
	av.SetKeyValueAttrs(proto.ValueOrDefault(avk))
	av.SetType(storage.Alert_Violation_K8S_EVENT)
	av.SetTime(event.GetTimestamp())
	return av
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
		avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
		avkk.SetKey(APIVerbKey)
		avkk.SetValue(event.GetApiVerb().String())
		attrs = append(attrs, avkk)
	}

	if event.GetUser() != nil {
		if event.GetUser().GetUsername() != "" {
			avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
			avkk.SetKey(UsernameKey)
			avkk.SetValue(event.GetUser().GetUsername())
			attrs = append(attrs, avkk)
		}
		if len(event.GetUser().GetGroups()) > 0 {
			avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
			avkk.SetKey(UserGroupsKey)
			avkk.SetValue(strings.Join(event.GetUser().GetGroups(), ", "))
			attrs = append(attrs, avkk)
		}
	}

	if event.GetUserAgent() != "" {
		avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
		avkk.SetKey(UserAgentKey)
		avkk.SetValue(event.GetUserAgent())
		attrs = append(attrs, avkk)
	}

	if len(event.GetSourceIps()) > 0 {
		avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
		avkk.SetKey(IPAddressKey)
		avkk.SetValue(strings.Join(event.GetSourceIps(), ", "))
		attrs = append(attrs, avkk)
	}

	if !options.skipResourceURI {
		if uriParts := strings.Split(event.GetRequestUri(), "?"); len(uriParts) > 0 && !stringutils.AllEmpty(uriParts...) {
			avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
			avkk.SetKey(ResourceURIKey)
			avkk.SetValue(uriParts[0])
			attrs = append(attrs, avkk)
		}
	}

	if event.GetImpersonatedUser() != nil {
		if event.GetImpersonatedUser().GetUsername() != "" {
			avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
			avkk.SetKey(ImpersonatedUsernameKey)
			avkk.SetValue(event.GetImpersonatedUser().GetUsername())
			attrs = append(attrs, avkk)
		}
		if len(event.GetImpersonatedUser().GetGroups()) > 0 {
			avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
			avkk.SetKey(ImpersonatedUserGroupsKey)
			avkk.SetValue(strings.Join(event.GetImpersonatedUser().GetGroups(), ", "))
			attrs = append(attrs, avkk)
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

func getExecMsgViolationAttr(event *storage.KubernetesEvent) []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr {
	attrs := make([]*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr, 0, 3)
	if pod := event.GetObject().GetName(); pod != "" {
		avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
		avkk.SetKey(PodKey)
		avkk.SetValue(pod)
		attrs = append(attrs, avkk)
	}

	args := event.GetPodExecArgs()
	if container := args.GetContainer(); container != "" {
		avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
		avkk.SetKey(ContainerKey)
		avkk.SetValue(container)
		attrs = append(attrs, avkk)
	}

	if cmds := stringutils.JoinNonEmpty(" ", args.GetCommands()...); cmds != "" {
		avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
		avkk.SetKey(CommandsKey)
		avkk.SetValue(cmds)
		attrs = append(attrs, avkk)
	}

	attrs = append(attrs, getDefaultViolationMsgViolationAttr(event, &attributeOptions{skipVerb: true, skipResourceURI: true})...)
	return attrs
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
	attrs := make([]*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr, 0, 2)
	if pod := event.GetObject().GetName(); pod != "" {
		avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
		avkk.SetKey(PodKey)
		avkk.SetValue(pod)
		attrs = append(attrs, avkk)
	}

	if ports := stringutils.JoinInt32(", ", event.GetPodPortForwardArgs().GetPorts()...); ports != "" {
		avkk := &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{}
		avkk.SetKey(PortsKey)
		avkk.SetValue(ports)
		attrs = append(attrs, avkk)
	}

	attrs = append(attrs, getDefaultViolationMsgViolationAttr(event, &attributeOptions{skipVerb: true, skipResourceURI: true})...)
	return attrs
}
