package proxy

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/pkg/utils"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/maputil"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileProxySecretExtension returns a reconcile extension that ensures that a proxy secret exists.
func ReconcileProxySecretExtension(client ctrlClient.Client, proxyEnv map[string]string) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), _ logr.Logger) error {
		if obj.GetDeletionTimestamp() != nil {
			return deleteProxyEnvSecret(ctx, obj, client)
		}

		return reconcileProxySecret(ctx, obj, proxyEnv, statusUpdater, client)
	}
}

func getProxyEnvSecretName(obj k8sutil.Object) string {
	return strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind + "-" + obj.GetName() + "-proxy-env")
}

func reconcileProxySecret(ctx context.Context, obj k8sutil.Object, proxyEnvVars map[string]string, statusUpdater func(extensions.UpdateStatusFunc), client ctrlClient.Client) error {
	var err error
	if len(proxyEnvVars) == 0 {
		err = deleteProxyEnvSecret(ctx, obj, client)
	} else {
		err = updateProxyEnvSecret(ctx, obj, client, proxyEnvVars)
	}

	if err != nil {
		statusUpdater(utils.UpdateStatusCondition(
			ProxyConfigFailedStatusType,
			metav1.ConditionTrue,
			SecretReconcileErrorReason,
			err.Error()))
		return nil // do not block reconciliation because of the proxy config
	}

	var reason, msg string
	if len(proxyEnvVars) == 0 {
		reason = NoProxyConfigReason
		msg = "No proxy configuration is desired"
	} else {
		reason = ProxyConfigAppliedReason
		msg = "Proxy configuration has been applied successfully"
	}
	statusUpdater(utils.UpdateStatusCondition(
		ProxyConfigFailedStatusType,
		metav1.ConditionFalse,
		reason,
		msg))
	return nil
}

func deleteProxyEnvSecret(ctx context.Context, obj k8sutil.Object, client ctrlClient.Client) error {
	existingSecret := &corev1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: obj.GetNamespace(), Name: getProxyEnvSecretName(obj)}
	if err := client.Get(ctx, key, existingSecret); err != nil {
		if apiErrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "error checking for proxy env secret")
	}

	if !metav1.IsControlledBy(existingSecret, obj) {
		return nil // don't touch a secret we don't own
	}

	return utils.DeleteExact(ctx, client, existingSecret)
}

func updateProxyEnvSecret(ctx context.Context, obj k8sutil.Object, client ctrlClient.Client, proxyEnvVars map[string]string) error {
	secretName := getProxyEnvSecretName(obj)

	secret := &corev1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: obj.GetNamespace(), Name: secretName}
	if err := client.Get(ctx, key, secret); err != nil {
		if !apiErrors.IsNotFound(err) {
			return err
		}
		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: obj.GetNamespace(),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(obj, obj.GetObjectKind().GroupVersionKind()),
				},
			},
		}
	} else if !metav1.IsControlledBy(secret, obj) {
		return errors.Errorf("secret %s exists, but is not controlled by %s", secretName, obj.GetName())
	}

	strData := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		strData[k] = string(v)
	}

	if maputil.Equal(strData, proxyEnvVars) {
		return nil
	}

	secret.Data = nil
	secret.StringData = proxyEnvVars

	if secret.ResourceVersion == "" {
		return client.Create(ctx, secret)
	}
	return client.Update(ctx, secret)
}
