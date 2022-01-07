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
		configmap, err := client.Get(ctx, name, metav1.GetOptions{})
		if err = validate(name, configmap, err); err != nil {
			return err
		}
		if configmap == nil {
			configmap = makeConfigMap(name, obj)
			takeControl(configmap, obj)
			if _, err := client.Create(ctx, configmap, metav1.CreateOptions{}); err != nil {
				return errors.Wrapf(err, "cannot create configMap %s", name)
			}
		} else if !metav1.IsControlledBy(configmap, obj) {
			takeControl(configmap, obj)
			if _, err := client.Update(ctx, configmap, metav1.UpdateOptions{}); err != nil {
				return errors.Wrapf(err, "cannot control configMap %s", name)
			}
		}
		return nil
	}
}

func validate(name string, configmap *corev1.ConfigMap, err error) error {
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "cannot retrieve configMap %s", name)
	}

	labels := configmap.GetLabels()
	if labels != nil {
		if v, ok := labels[label]; !ok || v != "true" {
			labels = nil
		}
	}
	if labels == nil {
		return errors.Errorf("configMap %s exists, but is not properly labeled", configmap.GetName())
	}
	return nil
}

func makeConfigMap(name string, controller *unstructured.Unstructured) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: controller.GetNamespace(),
			Labels: map[string]string{
				label: "true",
			},
		},
	}
}

func takeControl(configmap metav1.Object, controller *unstructured.Unstructured) {
	refs := configmap.GetOwnerReferences()
	if refs == nil {
		refs = []metav1.OwnerReference{}
	}
	newref := metav1.NewControllerRef(controller, controller.GroupVersionKind())
	configmap.SetOwnerReferences(append(refs, *newref))
}
