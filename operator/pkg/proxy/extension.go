package proxy

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/pkg/utils"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/maputil"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// ReconcileProxySecretExtension returns a reconcile extension that ensures that a proxy secret exists.
func ReconcileProxySecretExtension(k8sClient kubernetes.Interface, proxyEnv map[string]string) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), _ logr.Logger) error {
		secretsClient := k8sClient.CoreV1().Secrets(obj.GetNamespace())
		if obj.GetDeletionTimestamp() != nil {
			return deleteProxyEnvSecret(ctx, obj, secretsClient)
		}

		return reconcileProxySecret(ctx, obj, proxyEnv, statusUpdater, secretsClient)
	}
}

func getProxyEnvSecretName(obj k8sutil.Object) string {
	return strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind + "-" + obj.GetName() + "-proxy-env")
}

func reconcileProxySecret(ctx context.Context, obj k8sutil.Object, proxyEnvVars map[string]string, statusUpdater func(extensions.UpdateStatusFunc), secretsClient coreV1.SecretInterface) error {
	var err error
	if len(proxyEnvVars) == 0 {
		err = deleteProxyEnvSecret(ctx, obj, secretsClient)
	} else {
		err = updateProxyEnvSecret(ctx, obj, secretsClient, proxyEnvVars)
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

func deleteProxyEnvSecret(ctx context.Context, obj k8sutil.Object, secretsClient coreV1.SecretInterface) error {
	existingSecret, err := secretsClient.Get(ctx, getProxyEnvSecretName(obj), metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "error checking for proxy env secret")
	}

	if !metav1.IsControlledBy(existingSecret, obj) {
		return nil // don't touch a secret we don't own
	}

	return utils.DeleteExact(ctx, secretsClient, existingSecret)
}

func updateProxyEnvSecret(ctx context.Context, obj k8sutil.Object, secretsClient coreV1.SecretInterface, proxyEnvVars map[string]string) error {
	secretName := getProxyEnvSecretName(obj)

	secret, err := secretsClient.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
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

	if maputil.StringStringMapsEqual(strData, proxyEnvVars) {
		return nil
	}

	secret.Data = nil
	secret.StringData = proxyEnvVars

	if secret.ResourceVersion == "" {
		_, err := secretsClient.Create(ctx, secret, metav1.CreateOptions{})
		return err
	}
	_, err = secretsClient.Update(ctx, secret, metav1.UpdateOptions{})
	return err
}
