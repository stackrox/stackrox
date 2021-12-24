package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
)

const (
	configMapName = "injected-trusted-ca"
	annotation    = "config.openshift.io/inject-trusted-cabundle"
)

// InjectTrustedCAConfigMapExtension returns an extension that takes care of reconciling the injected trusted CA ConfigMap.
func InjectTrustedCAConfigMapExtension(client kubernetes.Interface) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), _ logr.Logger) error {

		cmClient := client.CoreV1().ConfigMaps(obj.GetNamespace())
		configmap, err := cmClient.Get(ctx, configMapName, metav1.GetOptions{})

		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "cannot retrieve configMap %s", configMapName)
		}
		found := err == nil

		if err = checkConfigMap(configmap, obj); err != nil {
			return err
		}

		if found && obj.GetDeletionTimestamp() != nil {
			return utils.DeleteExact(ctx, cmClient, configmap)

		} else if !found {
			if _, err = cmClient.Create(ctx, &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: obj.GetNamespace(),
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(obj, obj.GetObjectKind().GroupVersionKind()),
					},
					Annotations: map[string]string{
						annotation: "true",
					},
				}},
				metav1.CreateOptions{}); err != nil {
				return errors.Wrapf(err, "cannot create configMap %s", configMapName)
			}
		}

		return nil
	}
}

func checkConfigMap(configmap *corev1.ConfigMap, obj *unstructured.Unstructured) error {
	if configmap == nil {
		return nil
	}
	if !metav1.IsControlledBy(configmap, obj) {
		return errors.Errorf("configMap %s exists, but is not controlled by %s", configMapName, obj.GetName())
	}

	if v, ok := configmap.GetAnnotations()[annotation]; !ok || v != "true" {
		return errors.Errorf("configMap %s exists, but is not properly annotated", configMapName)
	}

	return nil
}
