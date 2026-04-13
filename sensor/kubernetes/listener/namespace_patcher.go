package listener

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/k8swatch"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

const (
	modifiedByAnnotation = `modified-by.stackrox.io/namespace-label-patcher`
)

func patchNamespaces(dynClient dynamic.Interface, stopCond concurrency.Waitable) {
	k8sClient := k8swatch.InClusterClient()
	nsInformer := k8swatch.NewInformerAdapter("/api/v1/namespaces", k8sClient, func() runtime.Object { return &v1.Namespace{} })

	patchHandler := &namespacePatchHandler{
		dynClient: dynClient,
		ctx:       concurrency.AsContext(stopCond),
	}

	if _, err := nsInformer.AddEventHandler(patchHandler); err != nil {
		log.Warnf("could not add event handler: %+v", err)
	}
	go nsInformer.Run(stopCond.Done())
}

type namespacePatchHandler struct {
	dynClient dynamic.Interface
	ctx       context.Context
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
	labels := make(map[string]string, len(ns.Labels)+len(desiredLabels))
	for k, v := range ns.Labels {
		labels[k] = v
	}
	for k, v := range desiredLabels {
		labels[k] = v
	}

	annotations := make(map[string]string, len(ns.Annotations)+1)
	for k, v := range ns.Annotations {
		annotations[k] = v
	}
	annotations[modifiedByAnnotation] = "true"

	patch, _ := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"labels":      labels,
			"annotations": annotations,
		},
	})

	_, err := h.dynClient.Resource(client.NamespaceGVR).Patch(h.ctx, ns.GetName(), k8sTypes.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return errors.Wrap(err, "patching namespace labels")
	}
	return nil
}
