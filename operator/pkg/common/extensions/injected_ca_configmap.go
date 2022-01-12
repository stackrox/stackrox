package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
				// Ignore missing ConfigMap.
				return nil
			}
			return errors.Wrapf(err, "cannot retrieve configMap %s", key.Name)
		}
		if err := validate(configMap); err != nil {
			return err
		}
		if controllerutil.SetControllerReference(obj, configMap, nil) == nil {
			if err := k8s.Update(ctx, configMap); err != nil {
				return errors.Wrapf(err, "cannot control configMap %s", key.Name)
			}
		}
		return nil
	}
}

func validate(configMap *corev1.ConfigMap) error {
	if configMap.GetLabels()["config.openshift.io/inject-trusted-cabundle"] != "true" {
		return errors.Errorf("configMap %s exists, but is not properly labeled", configMap.GetName())
	}
	return nil
}
