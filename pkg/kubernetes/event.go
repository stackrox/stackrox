package kubernetes

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/stringutils"
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
		return nil, errors.Errorf("currently do not recognize kind %q in admission controller", req.Kind)
	}
}

func podExecEvent(req *admission.AdmissionRequest) (*storage.KubernetesEvent, error) {
	apiVerb, supported := supportedAPIVerbs[req.Operation]
	if !supported {
		return nil, errors.Errorf("operation %s not supported", req.Operation)
	}

	var obj core.PodExecOptions
	if _, _, err := universalDeserializer.Decode(req.Object.Raw, nil, &obj); err != nil {
		return nil, err
	}

	return &storage.KubernetesEvent{
		Id:      string(req.UID),
		ApiVerb: apiVerb,
		Object: &storage.KubernetesEvent_Object{
			Name:      req.Name,
			Resource:  storage.KubernetesEvent_Object_PODS_EXEC,
			Namespace: req.Namespace,
		},
		ObjectArgs: &storage.KubernetesEvent_PodExecArgs_{
			PodExecArgs: &storage.KubernetesEvent_PodExecArgs{
				Container: obj.Container,
				Commands:  obj.Command,
			},
		},
		User: &storage.KubernetesEvent_User{
			Username: req.UserInfo.Username,
			Groups:   req.UserInfo.Groups,
		},
	}, nil
}

func podPortForwardEvent(req *admission.AdmissionRequest) (*storage.KubernetesEvent, error) {
	apiVerb, supported := supportedAPIVerbs[req.Operation]
	if !supported {
		return nil, errors.Errorf("operation %s not supported", req.Operation)
	}

	var obj core.PodPortForwardOptions
	if _, _, err := universalDeserializer.Decode(req.Object.Raw, nil, &obj); err != nil {
		return nil, err
	}

	return &storage.KubernetesEvent{
		Id:      string(req.UID),
		ApiVerb: apiVerb,
		Object: &storage.KubernetesEvent_Object{
			Name:      req.Name,
			Resource:  storage.KubernetesEvent_Object_PODS_PORTFORWARD,
			Namespace: req.Namespace,
		},
		ObjectArgs: &storage.KubernetesEvent_PodPortForwardArgs_{
			PodPortForwardArgs: &storage.KubernetesEvent_PodPortForwardArgs{
				Ports: obj.Ports,
			},
		},
		User: &storage.KubernetesEvent_User{
			Username: req.UserInfo.Username,
			Groups:   req.UserInfo.Groups,
		},
	}, nil
}
