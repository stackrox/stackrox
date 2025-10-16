package kubernetes

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/stringutils"
	"google.golang.org/protobuf/proto"
	admission "k8s.io/api/admission/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const (
	podExecOptionsKind        = "PodExecOptions"
	podPortForwardOptionsKind = "PodPortForwardOptions"
)

var (
	supportedAPIVerbs = map[admission.Operation]storage.KubernetesEvent_APIVerb{
		admission.Connect: storage.KubernetesEvent_CREATE,
	}

	universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()

	// ErrUnsupportedRequestKind is an error type indicating that we don't know how to handle an admission request.
	ErrUnsupportedRequestKind = errox.InvalidArgs.New("unsupported request kind")
	// ErrUnsupportedAPIVerb is an error type indicating that we don't know how to handle a certain operation in an
	// admission request.
	ErrUnsupportedAPIVerb = errox.InvalidArgs.New("unsupported API verb")
)

// EventAsString returns the kubernetes resources as string, such as, namespace/default/pod/nginx-86c57db685-nqq97/portforward.
func EventAsString(event *storage.KubernetesEvent) string {
	resource, subresource := stringutils.Split2(strings.ToLower(event.GetObject().GetResource().String()), "_")
	suffix := resource + "/" + event.GetObject().GetName()
	if subresource != "" {
		suffix = suffix + "/" + subresource
	}

	if event.GetObject().GetNamespace() == "" {
		return event.GetApiVerb().String() + ":" + suffix
	}

	return event.GetApiVerb().String() + ":" + "namespace/" + event.GetObject().GetNamespace() + "/" + suffix
}

// AdmissionRequestToKubeEventObj translates admission request into a kubernetes event object.
func AdmissionRequestToKubeEventObj(req *admission.AdmissionRequest) (*storage.KubernetesEvent, error) {
	switch req.Kind.Kind {
	case podExecOptionsKind:
		return podExecEvent(req)
	case podPortForwardOptionsKind:
		return podPortForwardEvent(req)
	default:
		return nil, ErrUnsupportedRequestKind.CausedByf("%q", req.Kind)
	}
}

func podExecEvent(req *admission.AdmissionRequest) (*storage.KubernetesEvent, error) {
	apiVerb, supported := supportedAPIVerbs[req.Operation]
	if !supported {
		return nil, ErrUnsupportedAPIVerb.CausedByf("%q", req.Operation)
	}

	var obj core.PodExecOptions
	if _, _, err := universalDeserializer.Decode(req.Object.Raw, nil, &obj); err != nil {
		return nil, err
	}

	ko := &storage.KubernetesEvent_Object{}
	ko.SetName(req.Name)
	ko.SetResource(storage.KubernetesEvent_Object_PODS_EXEC)
	ko.SetNamespace(req.Namespace)
	kp := &storage.KubernetesEvent_PodExecArgs{}
	kp.SetContainer(obj.Container)
	kp.SetCommands(obj.Command)
	ku := &storage.KubernetesEvent_User{}
	ku.SetUsername(req.UserInfo.Username)
	ku.SetGroups(req.UserInfo.Groups)
	ke := &storage.KubernetesEvent{}
	ke.SetId(string(req.UID))
	ke.SetApiVerb(apiVerb)
	ke.SetObject(ko)
	ke.SetPodExecArgs(proto.ValueOrDefault(kp))
	ke.SetUser(ku)
	return ke, nil
}

func podPortForwardEvent(req *admission.AdmissionRequest) (*storage.KubernetesEvent, error) {
	apiVerb, supported := supportedAPIVerbs[req.Operation]
	if !supported {
		return nil, ErrUnsupportedAPIVerb.CausedByf("%q", req.Operation)
	}

	var obj core.PodPortForwardOptions
	if _, _, err := universalDeserializer.Decode(req.Object.Raw, nil, &obj); err != nil {
		return nil, err
	}

	ko := &storage.KubernetesEvent_Object{}
	ko.SetName(req.Name)
	ko.SetResource(storage.KubernetesEvent_Object_PODS_PORTFORWARD)
	ko.SetNamespace(req.Namespace)
	kp := &storage.KubernetesEvent_PodPortForwardArgs{}
	kp.SetPorts(obj.Ports)
	ku := &storage.KubernetesEvent_User{}
	ku.SetUsername(req.UserInfo.Username)
	ku.SetGroups(req.UserInfo.Groups)
	ke := &storage.KubernetesEvent{}
	ke.SetId(string(req.UID))
	ke.SetApiVerb(apiVerb)
	ke.SetObject(ko)
	ke.SetPodPortForwardArgs(proto.ValueOrDefault(kp))
	ke.SetUser(ku)
	return ke, nil
}
