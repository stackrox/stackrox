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
	label         = "config.openshift.io/inject-trusted-cabundle"
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
		case takeControl(configmap, obj):
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
	labels := configmap.GetLabels()
	if labels != nil {
		if v, ok := labels[label]; !ok || v != "true" {
			labels = nil
		}
	}
	if labels == nil {
		return nil, errors.Errorf("configMap %s exists, but is not properly labeled", configMapName)
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
			Labels: map[string]string{
				label: "true",
			},
		}}
}

// takeControl returns true if the provided configmap structure has been altered and therefore
// needs to be updated in the cluster.
func takeControl(configmap *corev1.ConfigMap, obj *unstructured.Unstructured) bool {
	controller := getController(configmap)

	// Don't alter configmap if it's already properly controlled.
	if controller != nil &&
		(controller.Kind == "Central" && obj.GetKind() == "SecuredCluster" ||
			controller.UID == obj.GetUID()) {
		return false
	}

	not := false

	// Remove control from the existing controller, but keep it as an owner.
	controller.Controller = &not
	controller.BlockOwnerDeletion = &not

	// Add new controller.
	newref := metav1.NewControllerRef(obj, obj.GroupVersionKind())

	// Allow k8s garbage collector to delete the owner if needed.
	newref.BlockOwnerDeletion = &not

	refs := configmap.GetOwnerReferences()
	if refs == nil {
		refs = []metav1.OwnerReference{}
	}
	configmap.SetOwnerReferences(append(refs, *newref))
	return true
}

func getController(configmap *corev1.ConfigMap) *metav1.OwnerReference {
	refs := configmap.GetOwnerReferences()
	for i := range refs {
		if refs[i].Controller != nil && *refs[i].Controller {
			return &refs[i]
		}
	}
	return nil
}
