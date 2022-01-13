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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// InjectTrustedCAConfigMapExtension returns an extension that takes care of reconciling the injected trusted CA ConfigMap.
func InjectTrustedCAConfigMapExtension(k8s client.Client) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), _ logr.Logger) error {
		configMap := &corev1.ConfigMap{}
		key := client.ObjectKey{Namespace: obj.GetNamespace(), Name: "injected-cabundle-" + obj.GetName()}
		if err := k8s.Get(ctx, key, configMap); err != nil {
			if apiErrors.IsNotFound(err) {
				configMap = makeConfigMap(&key)
				if err := controllerutil.SetControllerReference(obj, configMap, nil); err != nil {
					return errors.Wrapf(err, "cannot set configMap %s controller", key.Name)
				}
				if err := k8s.Create(ctx, configMap); err != nil {
					return errors.Wrapf(err, "cannot create configMap %s", key.Name)
				}
				return nil
			}
			return errors.Wrapf(err, "cannot retrieve configMap %s", key.Name)
		}
		return nil
	}
}

func makeConfigMap(key *types.NamespacedName) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
			Labels: map[string]string{
				"config.openshift.io/inject-trusted-cabundle": "true",
			},
		},
	}
}
