package listener

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/namespaces"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	modifiedByAnnotation = `modified-by.stackrox.io/namespace-label-patcher`
)

func patchNamespaces(client kubernetes.Interface, stopCond concurrency.Waitable) {
	nsInformer := informers.NewSharedInformerFactory(client, resyncPeriod).Core().V1().Namespaces().Informer()
	nsClient := client.CoreV1().Namespaces()

	patchHandler := &namespacePatchHandler{
		nsClient: nsClient,
	}
	nsInformer.AddEventHandler(patchHandler)
	go nsInformer.Run(stopCond.Done())
}

type namespacePatchHandler struct {
	nsClient coreV1.NamespaceInterface
}

func (h *namespacePatchHandler) OnAdd(obj interface{}) {
	h.checkAndPatchNamespace(obj)
}

func (h *namespacePatchHandler) OnUpdate(oldObj, newObj interface{}) {
	h.checkAndPatchNamespace(newObj)
}

func (h *namespacePatchHandler) OnDelete(obj interface{}) {}

func checkDesiredLabels(actual, desired map[string]string) bool {
	for k, v := range desired {
		if actual[k] != v {
			return false
		}
	}
	return true
}

func (h *namespacePatchHandler) checkAndPatchNamespace(obj interface{}) {
	ns, ok := obj.(*v1.Namespace)
	if !ok {
		return
	}

	desiredLabels := map[string]string{
		namespaces.NamespaceIDLabel:   string(ns.GetUID()),
		namespaces.NamespaceNameLabel: ns.GetName(),
	}

	if checkDesiredLabels(ns.GetLabels(), desiredLabels) {
		return
	}

	if err := h.patchNamespaceLabels(ns, desiredLabels); err != nil {
		// No need to retry because of concurrenct updates - in this case, we'll process another event for this object
		// anyway.
		log.Errorf("patching namespace %s: %v", ns.GetName(), err)
	}
}

func (h *namespacePatchHandler) patchNamespaceLabels(ns *v1.Namespace, desiredLabels map[string]string) error {
	patchedNS := ns.DeepCopy()
	if patchedNS.Labels == nil {
		patchedNS.Labels = desiredLabels
	} else {
		for k, v := range desiredLabels {
			patchedNS.Labels[k] = v
		}
	}

	if patchedNS.Annotations == nil {
		patchedNS.Annotations = make(map[string]string)
	}
	patchedNS.Annotations[modifiedByAnnotation] = "true"

	_, err := h.nsClient.Update(patchedNS)
	return err
}
