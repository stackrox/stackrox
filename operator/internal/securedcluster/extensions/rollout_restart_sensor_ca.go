package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	commonAnnotations "github.com/stackrox/rox/operator/internal/common/annotations"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/pkg/crs"
	"github.com/stackrox/rox/pkg/securedcluster"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var allTLSSecretNames = []string{
	securedcluster.CollectorTLSSecretName,
	securedcluster.AdmissionControlTLSSecretName,
	securedcluster.ScannerTLSSecretName,
	securedcluster.ScannerDbTLSSecretName,
	securedcluster.ScannerV4IndexerTLSSecretName,
	securedcluster.ScannerV4DbTLSSecretName,
}

// RolloutRestartOnSensorCAChange is an extension that triggers rollout restarts of workloads
// when the Sensor CA changes.
func RolloutRestartOnSensorCAChange(client ctrlClient.Client, direct ctrlClient.Reader, logger logr.Logger) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), log logr.Logger) error {
		logger = logger.WithName("rollout-restart-sensor-ca")

		sensorHash, fromRuntimeSecret, err := tryGetSensorCAHash(ctx, client, direct, obj.GetNamespace())
		if err != nil {
			return err
		}

		annotations := getOrInitAnnotations(obj)

		// runtime TLS secrets are not stored atomically, so we should wait for consistency
		if fromRuntimeSecret {
			if err := waitForCAConsistency(ctx, client, direct, obj.GetNamespace(), sensorHash, logger); err != nil {
				return err
			}
		}

		// persist new annotation in-memory (for PostRenderer)
		annotations[commonAnnotations.ConfigHashAnnotation] = sensorHash
		obj.SetAnnotations(annotations)

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

	return "", false, errors.New("no TLS secrets found: tls-cert-sensor, cluster-registration-secret, or sensor-tls")
}

func getOrInitAnnotations(obj *unstructured.Unstructured) map[string]string {
	anns := obj.GetAnnotations()
	if anns == nil {
		anns = make(map[string]string)
	}
	return anns
}

// verifyAllTLSSecretsMatchCA checks that all TLS secrets have the given ca.pem hash
// Returns (isConsistent, error)
func verifyAllTLSSecretsMatchCA(ctx context.Context, client ctrlClient.Client, direct ctrlClient.Reader, namespace string, expectedCAHash string, logger logr.Logger) (bool, error) {
	for _, secretName := range allTLSSecretNames {
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

// waitForCAConsistency waits until all TLS runtime secrets have the expected CA cert hash.
func waitForCAConsistency(ctx context.Context, client ctrlClient.Client, direct ctrlClient.Reader, namespace string, expectedHash string, logger logr.Logger) error {

	consistentDeadline := 10 * time.Second
	pollInterval := 500 * time.Millisecond

	waitCtx, cancel := context.WithTimeout(ctx, consistentDeadline)
	defer cancel()

	for {
		select {
		case <-waitCtx.Done():
			logger.Info("Timed out waiting for TLS secrets to reach consistent CA; will retry on next reconcile")
			return nil
		default:
		}

		consistent, err := verifyAllTLSSecretsMatchCA(waitCtx, client, direct, namespace, expectedHash, logger)
		if err != nil {
			return errors.Wrap(err, "failed to check TLS secrets CA consistency")
		}
		if consistent {
			return nil
		}
		time.Sleep(pollInterval)
	}
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
	sum := sha256.Sum256(caPEM)
	return hex.EncodeToString(sum[:]), nil
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

	sum := sha256.Sum256([]byte(parsedCRS.CAs[0]))
	return hex.EncodeToString(sum[:]), nil
}
