package vmhelpers

import (
	"context"
	"fmt"
	"os"
	"time"

	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

const (
	// ImagePullSecretName is the name of the k8s Secret used to pull VM guest
	// images from a private registry.
	ImagePullSecretName = "vm-image-pull-secret" //nolint:gosec // G101: not a credential, just the k8s Secret resource name

	// defaultServiceAccountPollInterval is the polling cadence while waiting
	// for the default ServiceAccount to appear in a freshly-created namespace.
	defaultServiceAccountPollInterval = 2 * time.Second

	// defaultServiceAccountWaitTimeout is the ceiling for waiting on the
	// default service account to appear in a namespace.
	defaultServiceAccountWaitTimeout = 10 * time.Second
)

// EnsureImagePullSecret creates or updates a docker-config-json image pull
// secret in ns from the docker config JSON at secretPath, and links it to the
// namespace's default ServiceAccount. It is a no-op when secretPath is empty.
func EnsureImagePullSecret(ctx context.Context, k8sClient kubernetes.Interface, logf func(string, ...any), ns, secretName, secretPath string) error {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if secretPath == "" {
		return nil
	}

	logf("provision VMs: creating image pull secret from %q", secretPath)
	dockerCfg, err := os.ReadFile(secretPath)
	if err != nil {
		return fmt.Errorf("read image pull secret file %q: %w", secretPath, err)
	}

	secret := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{Name: secretName},
		Type:       coreV1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{coreV1.DockerConfigJsonKey: dockerCfg},
	}
	_, err = k8sClient.CoreV1().Secrets(ns).Create(ctx, secret, metaV1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		var existingSecret *coreV1.Secret
		existingSecret, err = k8sClient.CoreV1().Secrets(ns).Get(ctx, secretName, metaV1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get existing image pull secret %q in namespace %q: %w", secretName, ns, err)
		}
		existingSecret.Type = coreV1.SecretTypeDockerConfigJson
		if existingSecret.Data == nil {
			existingSecret.Data = make(map[string][]byte)
		}
		existingSecret.Data[coreV1.DockerConfigJsonKey] = dockerCfg
		_, err = k8sClient.CoreV1().Secrets(ns).Update(ctx, existingSecret, metaV1.UpdateOptions{})
	}
	if err != nil {
		return fmt.Errorf("ensure image pull secret %q in namespace %q: %w", secretName, ns, err)
	}

	// Wait for the default SA to exist before attempting the update.
	if _, err = waitForDefaultServiceAccount(ctx, k8sClient, ns); err != nil {
		return fmt.Errorf("wait for default service account in namespace %q: %w", ns, err)
	}

	// The SA controller may still be mutating the freshly-created default SA
	// (e.g. patching token secrets), so a single Get+Update can hit an
	// optimistic concurrency conflict. Retry exactly like the DaemonSet
	// update in EnsureComplianceMetricsEnv.
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		sa, getErr := k8sClient.CoreV1().ServiceAccounts(ns).Get(ctx, "default", metaV1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		for _, ref := range sa.ImagePullSecrets {
			if ref.Name == secretName {
				return nil
			}
		}
		sa.ImagePullSecrets = append(sa.ImagePullSecrets, coreV1.LocalObjectReference{Name: secretName})
		_, updateErr := k8sClient.CoreV1().ServiceAccounts(ns).Update(ctx, sa, metaV1.UpdateOptions{})
		return updateErr
	})
	if err != nil {
		return fmt.Errorf("link image pull secret to default service account in namespace %q: %w", ns, err)
	}
	logf("provision VMs: image pull secret %q ready in namespace %q", secretName, ns)
	return nil
}

func waitForDefaultServiceAccount(ctx context.Context, k8sClient kubernetes.Interface, ns string) (*coreV1.ServiceAccount, error) {
	waitCtx, cancel := context.WithTimeout(ctx, defaultServiceAccountWaitTimeout)
	defer cancel()

	var serviceAccount *coreV1.ServiceAccount
	err := wait.PollUntilContextCancel(waitCtx, defaultServiceAccountPollInterval, true, func(ctx context.Context) (bool, error) {
		sa, err := k8sClient.CoreV1().ServiceAccounts(ns).Get(ctx, "default", metaV1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		serviceAccount = sa
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return serviceAccount, nil
}
