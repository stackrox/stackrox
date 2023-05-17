package listener

import (
	"context"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/namespaces"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	modifiedByAnnotation = `modified-by.stackrox.io/namespace-label-patcher`
)

func patchNamespaces(client kubernetes.Interface, stopCond concurrency.Waitable) {
	nsInformer := informers.NewSharedInformerFactory(client, noResyncPeriod).Core().V1().Namespaces().Informer()
	nsClient := client.CoreV1().Namespaces()

	patchHandler := &namespacePatchHandler{
		nsClient: nsClient,
		ctx:      concurrency.AsContext(stopCond),
	}

	if _, err := nsInformer.AddEventHandler(patchHandler); err != nil {
		log.Warnf("could not add event handler: %+v", err)
	}
	go nsInformer.Run(stopCond.Done())
}

type namespacePatchHandler struct {
	nsClient coreV1.NamespaceInterface
	ctx      context.Context
}

func (h *namespacePatchHandler) OnAdd(obj interface{}, _ bool) {
	h.checkAndPatchNamespace(obj)
}

func (h *namespacePatchHandler) OnUpdate(_, newObj interface{}) {
	h.checkAndPatchNamespace(newObj)
}

func (h *namespacePatchHandler) OnDelete(_ interface{}) {}

func (h *namespacePatchHandler) checkAndPatchNamespace(obj interface{}) {
	ns, ok := obj.(*v1.Namespace)
	if !ok {
		return
	}

	key := namespaces.GetFirstValidNamespaceNameLabelKey(ns.GetLabels(), ns.GetName())
	if key != "" {
		return
	}

	desiredLabels := map[string]string{
		namespaces.NamespaceNameLabel: ns.GetName(),
	}
	if err := h.patchNamespaceLabels(ns, desiredLabels); err != nil {
		// No need to retry because of concurrent updates - in this case, we'll process another event for this object
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

	_, err := h.nsClient.Update(h.ctx, patchedNS, metav1.UpdateOptions{})
	return err
}
