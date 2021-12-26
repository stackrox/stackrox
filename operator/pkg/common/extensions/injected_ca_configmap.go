package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	configMapName = "injected-trusted-ca"
	annotation    = "config.openshift.io/inject-trusted-cabundle"
)

// InjectTrustedCAConfigMapExtension returns an extension that takes care of reconciling the injected trusted CA ConfigMap.
func InjectTrustedCAConfigMapExtension(k8s kubernetes.Interface) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), _ logr.Logger) error {
		client := k8s.CoreV1().ConfigMaps(obj.GetNamespace())
		switch configmap, err := getConfigMap(ctx, client, obj); {
		case err != nil:
			return err
		case configmap == nil:
			if _, err := client.Create(ctx, makeConfigMap(obj), metav1.CreateOptions{}); err != nil {
				return errors.Wrapf(err, "cannot create configMap %s", configMapName)
			}
		case !isAlreadyControlled(configmap, obj):
			addController(configmap, obj)
			if _, err := client.Update(ctx, configmap, metav1.UpdateOptions{}); err != nil {
				return errors.Wrapf(err, "cannot control configMap %s", configMapName)
			}
		}
		return nil
	}
}

func getConfigMap(ctx context.Context, client v1.ConfigMapInterface, obj *unstructured.Unstructured) (*corev1.ConfigMap, error) {
	configmap, err := client.Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "cannot retrieve configMap %s", configMapName)
	}
	annotations := configmap.GetAnnotations()
	if annotations != nil {
		if v, ok := annotations[annotation]; !ok || v != "true" {
			annotations = nil
		}
	}
	if annotations == nil {
		return nil, errors.Errorf("configMap %s exists, but is not properly annotated", configMapName)
	}

	return configmap, nil
}

func makeConfigMap(controller *unstructured.Unstructured) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: controller.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(controller, controller.GroupVersionKind()),
			},
			Annotations: map[string]string{
				annotation: "true",
			},
		}}
}

func addController(configmap *corev1.ConfigMap, obj *unstructured.Unstructured) {
	refs := configmap.GetOwnerReferences()
	if refs == nil {
		refs = []metav1.OwnerReference{}
	}
	newref := metav1.NewControllerRef(obj, obj.GroupVersionKind())
	blockOwnerDeletion := false
	newref.BlockOwnerDeletion = &blockOwnerDeletion
	refs = append(refs, *newref)
	configmap.SetOwnerReferences(refs)
}

func isAlreadyControlled(configmap *corev1.ConfigMap, obj *unstructured.Unstructured) bool {
	if obj.GetUID() == "" {
		return false
	}
	refs := configmap.GetOwnerReferences()
	for i := range refs {
		if refs[i].Controller != nil && refs[i].UID == obj.GetUID() {
			return true
		}
	}
	return false
}
