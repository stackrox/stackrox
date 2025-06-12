package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CABundleConfigMapName = "tls-ca-bundle"

	validatingWebhookConfigName = "stackrox"
	caBundleConfigMapKey        = "ca-bundle.pem"
)

// ReconcileAdmissionControlCABundleExtension returns an extension that synchronizes the
// ValidatingWebhookConfiguration's caBundle with the ConfigMap managed by Sensor.
func ReconcileAdmissionControlCABundleExtension(client ctrlClient.Client, direct ctrlClient.Reader) extensions.ReconcileExtension {
	return wrapExtension(reconcileCABundle, client, direct)
}

func reconcileCABundle(ctx context.Context, sc *platform.SecuredCluster, client ctrlClient.Client, directClient ctrlClient.Reader,
	_ func(updateStatusFunc), log logr.Logger) error {
	run := &reconcileCABundleRun{
		client:         client,
		directClient:   directClient,
		securedCluster: sc,
		log:            log.WithName("validation-webhook-ca-bundle"),
	}
	return run.Execute(ctx)
}

type reconcileCABundleRun struct {
	client         ctrlClient.Client
	directClient   ctrlClient.Reader
	securedCluster *platform.SecuredCluster
	log            logr.Logger
}

func (r *reconcileCABundleRun) Execute(ctx context.Context) error {
	var caBundle corev1.ConfigMap
	key := ctrlClient.ObjectKey{
		Namespace: r.securedCluster.GetNamespace(),
		Name:      CABundleConfigMapName,
	}

	// use the direct client because the Operator does not manage this resource
	if err := r.directClient.Get(ctx, key, &caBundle); err != nil {
		if k8sErrors.IsNotFound(err) {
			r.log.Info("CA bundle ConfigMap not found, skipping caBundle update.",
				"namespace", r.securedCluster.GetNamespace(), "name", CABundleConfigMapName)
			return nil // Sensor may not have created the ConfigMap yet
		}
		return errors.Wrapf(err, "failed to get CA bundle ConfigMap %s", key)
	}

	caBundlePEM, ok := caBundle.Data[caBundleConfigMapKey]
	if !ok || len(caBundlePEM) == 0 {
		r.log.Info("Key is missing or empty in the CA bundle ConfigMap, skipping.")
		return nil // Not an error if the ConfigMap is present but empty.
	}

	var webhookConfig admissionv1.ValidatingWebhookConfiguration
	webhookConfigKey := ctrlClient.ObjectKey{Name: validatingWebhookConfigName}
	if err := r.client.Get(ctx, webhookConfigKey, &webhookConfig); err != nil {
		if k8sErrors.IsNotFound(err) {
			r.log.Info("ValidatingWebhookConfiguration not found, cannot update caBundle.")
			return nil
		}
		return errors.Wrapf(err, "failed to get ValidatingWebhookConfiguration %s", validatingWebhookConfigName)
	}

	return r.ensureCABundleIsUpToDate(ctx, &webhookConfig, caBundlePEM)
}

func (r *reconcileCABundleRun) ensureCABundleIsUpToDate(ctx context.Context, webhookConfig *admissionv1.ValidatingWebhookConfiguration, caBundlePEM string) error {
	for i := range webhookConfig.Webhooks {
		webhookConfig.Webhooks[i].ClientConfig.CABundle = []byte(caBundlePEM)
	}

	if err := r.client.Update(ctx, webhookConfig); err != nil {
		return errors.Wrapf(err, "failed to update ValidatingWebhookConfiguration %s", webhookConfig.Name)
	}

	return nil
}
