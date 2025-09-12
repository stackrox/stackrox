package extensions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/internal/common/confighash"
	"github.com/stackrox/rox/operator/internal/common/rendercache"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/pkg/crs"
	"github.com/stackrox/rox/pkg/securedcluster"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SensorCAHashExtension is an extension that computes and caches the CA hash for Secured Clusters,
// enabling declarative rollout restarts when the CA changes.
func SensorCAHashExtension(client ctrlClient.Client, direct ctrlClient.Reader, logger logr.Logger, renderCache *rendercache.RenderCache) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), log logr.Logger) error {
		logger = logger.WithName("sensor-ca-hash")

		// Clean up render cache entry if CR is being deleted.
		if obj.GetDeletionTimestamp() != nil {
			renderCache.Delete(obj)
			return nil
		}

		sensorHash, fromRuntimeSecret, err := tryGetSensorCAHash(ctx, client, direct, obj.GetNamespace())
		if err != nil {
			return err
		}

		// Runtime TLS secrets are not stored atomically, check if they are consistent.
		if fromRuntimeSecret {
			secretsConsistent, err := verifyAllTLSSecretsMatchCA(ctx, client, direct, obj.GetNamespace(), sensorHash, logger)
			if err != nil {
				return err
			}
			if !secretsConsistent {
				return errors.New("runtime-retrieved TLS secrets (`tls-cert-*`) are not consistent (are signed by different CAs), see operator log for details")
			}
		}

		// Store the CA hash in the render cache for the post renderer.
		renderCache.SetCAHash(obj, sensorHash)

		return nil
	}
}

// tryGetSensorCAHash attempts to get the sensor CA hash from different sources, in this order:
// 1. tls-cert-sensor (preferred, retrieved by Sensor from Central at runtime)
// 2. cluster-registration-secret (CRS token, used for Secured Cluster initialization)
// 3. sensor-tls (legacy init bundle fallback)
// Returns (hash, fromRuntimeSecret, error) where fromRuntimeSecret indicates if the hash came from tls-cert-sensor
func tryGetSensorCAHash(ctx context.Context, client ctrlClient.Client, direct ctrlClient.Reader, namespace string) (string, bool, error) {
	caSources := []struct {
		name            string
		isRuntimeSecret bool
		getCAHash       func() (string, error)
	}{
		{
			name:            "tls-cert-sensor",
			isRuntimeSecret: true,
			getCAHash: func() (string, error) {
				return hashSecretCA(ctx, client, direct, namespace, securedcluster.SensorTLSSecretName)
			},
		},
		{
			name:            "cluster-registration-secret",
			isRuntimeSecret: false,
			getCAHash:       func() (string, error) { return hashCRSCA(ctx, client, direct, namespace) },
		},
		{
			name:            "sensor-tls",
			isRuntimeSecret: false,
			getCAHash:       func() (string, error) { return hashSecretCA(ctx, client, direct, namespace, "sensor-tls") },
		},
	}

	for _, caSource := range caSources {
		hash, err := caSource.getCAHash()
		if err == nil {
			return hash, caSource.isRuntimeSecret, nil
		}
		if !k8sErrors.IsNotFound(err) {
			return "", false, errors.Wrapf(err, "failed to get sensor CA hash from %s", caSource.name)
		}
	}

	return "", false, fmt.Errorf("some init-bundle secrets missing in namespace %q, "+
		"please make sure you have downloaded init-bundle secrets (from UI or with roxctl) "+
		"and created corresponding resources in the correct namespace", namespace)
}

// verifyAllTLSSecretsMatchCA checks that all TLS secrets have the given ca.pem hash
// Returns (isConsistent, error)
func verifyAllTLSSecretsMatchCA(ctx context.Context, client ctrlClient.Client, direct ctrlClient.Reader, namespace string, expectedCAHash string, logger logr.Logger) (bool, error) {
	for _, secretName := range securedcluster.AllTLSSecretNames {
		caHash, err := hashSecretCA(ctx, client, direct, namespace, secretName)
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				logger.Info("Secret not found", "secret", secretName)
				return false, nil
			}

			return false, errors.Wrapf(err, "failed to get secret %s", secretName)
		}

		if expectedCAHash != caHash {
			logger.Info("CA hash mismatch detected",
				"secret", secretName,
				"expected-hash", expectedCAHash,
				"actual-hash", caHash)
			return false, nil
		}
	}

	return true, nil
}

// hashSecretCA reads the named secret and returns (hex(sha256(ca.pem)), error)
func hashSecretCA(ctx context.Context, client ctrlClient.Client, direct ctrlClient.Reader, namespace, secretName string) (string, error) {
	var secret corev1.Secret
	key := types.NamespacedName{Namespace: namespace, Name: secretName}
	if err := utils.GetWithFallbackToUncached(ctx, client, direct, key, &secret); err != nil {
		return "", err
	}
	caPEM := secret.Data["ca.pem"]
	if len(caPEM) == 0 {
		return "", fmt.Errorf("ca.pem is empty in %s secret", secretName)
	}
	return confighash.ComputeCAHash(caPEM), nil
}

// hashCRSCA reads the cluster-registration-secret and returns (hex(sha256(first_ca)), error)
func hashCRSCA(ctx context.Context, client ctrlClient.Client, direct ctrlClient.Reader, namespace string) (string, error) {
	var secret corev1.Secret
	key := types.NamespacedName{Namespace: namespace, Name: "cluster-registration-secret"}
	if err := utils.GetWithFallbackToUncached(ctx, client, direct, key, &secret); err != nil {
		return "", err
	}

	crsData := secret.Data["crs"]
	if len(crsData) == 0 {
		return "", errors.New("crs data is empty in cluster-registration-secret")
	}

	parsedCRS, err := crs.DeserializeSecret(string(crsData))
	if err != nil {
		return "", errors.Wrap(err, "deserializing CRS")
	}

	if len(parsedCRS.CAs) == 0 {
		return "", errors.New("no CAs found in CRS")
	}

	return confighash.ComputeCAHash([]byte(parsedCRS.CAs[0])), nil
}
