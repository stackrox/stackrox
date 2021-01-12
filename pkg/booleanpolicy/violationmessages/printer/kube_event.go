package printer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	defaultKubeEventViolationHeader = "Kubernetes event detected"
)

var (
	kubeEventViolationHeaderMsgs = map[kubeOpIndicator]string{
		{
			resource: storage.KubernetesEvent_Object_PODS_EXEC,
			apiVerb:  storage.KubernetesEvent_CREATE,
		}: "Kubectl exec into pod",
		{
			resource: storage.KubernetesEvent_Object_PODS_PORTFORWARD,
			apiVerb:  storage.KubernetesEvent_CREATE,
		}: "Kubectl port-forward into pod",
	}
)

type kubeOpIndicator struct {
	resource storage.KubernetesEvent_Object_Resource
	apiVerb  storage.KubernetesEvent_APIVerb
}

// GenerateKubeEventViolationMsg constructs violation message for kubernetes event violations.
func GenerateKubeEventViolationMsg(kubeEvent *storage.KubernetesEvent) *storage.Alert_Violation {
	msg := kubeEventViolationHeaderMsgs[kubeOpIndicator{
		resource: kubeEvent.GetObject().GetResource(),
		apiVerb:  kubeEvent.GetApiVerb(),
	}]

	if msg == "" {
		utils.Should(errors.Errorf("violation header message not defined for %s", kubernetes.EventAsString(kubeEvent)))
		msg = defaultKubeEventViolationHeader
	}

	return &storage.Alert_Violation{
		Message: msg,
		// TODO: fill the message attributes
	}
}
