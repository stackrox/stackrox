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
	configMapNamePrefix = "injected-cabundle-"
	label               = "config.openshift.io/inject-trusted-cabundle"
)

// InjectTrustedCAConfigMapExtension returns an extension that takes care of reconciling the injected trusted CA ConfigMap.
func InjectTrustedCAConfigMapExtension(k8s kubernetes.Interface) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), _ logr.Logger) error {
		client := k8s.CoreV1().ConfigMaps(obj.GetNamespace())
		name := configMapNamePrefix + obj.GetName()
		switch configmap, err := getConfigMap(ctx, name, client, obj); {
		case err != nil:
			return err
		case configmap == nil:
			if _, err := client.Create(ctx, makeConfigMap(obj), metav1.CreateOptions{}); err != nil {
				return errors.Wrapf(err, "cannot create configMap %s", name)
			}
		case takeControl(configmap, obj):
			if _, err := client.Update(ctx, configmap, metav1.UpdateOptions{}); err != nil {
				return errors.Wrapf(err, "cannot control configMap %s", name)
			}
		}
		return nil
	}
}

func getConfigMap(ctx context.Context, name string, client v1.ConfigMapInterface, obj *unstructured.Unstructured) (*corev1.ConfigMap, error) {
	configmap, err := client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "cannot retrieve configMap %s", name)
	}
	labels := configmap.GetLabels()
	if labels != nil {
		if v, ok := labels[label]; !ok || v != "true" {
			labels = nil
		}
	}
	if labels == nil {
		return nil, errors.Errorf("configMap %s exists, but is not properly labeled", name)
	}

	return configmap, nil
}

func makeConfigMap(controller *unstructured.Unstructured) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapNamePrefix,
			Namespace: controller.GetNamespace(),
			Labels: map[string]string{
				label: "true",
			},
		}}
	takeControl(cm, controller)
	return cm
}

// takeControl returns true if the provided object has been altered and therefore
// needs to be updated in the cluster.
func takeControl(obj metav1.Object, newController *unstructured.Unstructured) bool {
	not := false

	if currentController := metav1.GetControllerOfNoCopy(obj); currentController != nil {

		// Don't alter the object if it's already properly controlled.
		if currentController.UID == newController.GetUID() ||
			currentController.Kind == "Central" && newController.GetKind() == "SecuredCluster" ||
			newController.GetDeletionTimestamp() != nil {
			return false
		}

		// Remove control from the existing controller, but keep it as an owner.
		currentController.Controller = &not
		currentController.BlockOwnerDeletion = &not
	}

	// Add new controller.
	newref := metav1.NewControllerRef(newController, newController.GroupVersionKind())

	// Allow k8s garbage collector to delete the owner if needed.
	newref.BlockOwnerDeletion = &not

	refs := obj.GetOwnerReferences()
	if refs == nil {
		refs = []metav1.OwnerReference{}
	}
	obj.SetOwnerReferences(append(refs, *newref))
	return true
}
