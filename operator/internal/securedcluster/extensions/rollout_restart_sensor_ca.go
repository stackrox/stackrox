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
	common "github.com/stackrox/rox/operator/internal/common"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/pkg/securedcluster"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// sensorCAHashAnnotation is an annotation added to the Secured Cluster CR, used to store the hash of the Sensor CA
	// This is used to detect changes in the Sensor CA and trigger rollout restarts of workloads
	sensorCAHashAnnotation = "stackrox.io/sensor-ca-hash"
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

		// Read current sensor TLS secret hash
		sensorHash, err := hashSecretCA(ctx, client, direct, obj.GetNamespace(), securedcluster.SensorTLSSecretName)
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				// Secret doesn't exist yet - normal after fresh install
				return nil
			}
			return errors.Wrap(err, "failed to get sensor CA hash")
		}

		// Get the stored hash from the Secured Cluster CR annotations
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}

		storedHash := annotations[sensorCAHashAnnotation]
		if storedHash == "" {
			// Patch the SecuredCluster CR to persist the annotation
			patch := ctrlClient.MergeFrom(obj.DeepCopyObject().(ctrlClient.Object))
			annotations[sensorCAHashAnnotation] = sensorHash
			obj.SetAnnotations(annotations)
			if err := client.Patch(ctx, obj, patch); err != nil {
				return errors.Wrap(err, "failed to patch SecuredCluster with sensor CA hash")
			}

			// No stored hash yet - this is a fresh install or first time seeing the secret
			logger.Info("Stored sensor CA hash in SecuredCluster annotations", "hash", sensorHash)
			return nil
		}

		if storedHash == sensorHash {
			return nil
		}

		// Sensor secret changed - wait briefly for all TLS secrets to become consistent
		// This avoids restarting pods while different components have different CAs
		consistentDeadline := 10 * time.Second
		pollInterval := 500 * time.Millisecond

		var consistent bool
		waitCtx, cancel := context.WithTimeout(ctx, consistentDeadline)
		defer cancel()
		for {
			select {
			case <-waitCtx.Done():
				logger.Info("Timed out waiting for TLS secrets to reach consistent CA; will retry on next reconcile")
				return nil
			default:
			}

			var errConsistency error
			consistent, errConsistency = verifyAllTLSSecretsMatchCA(waitCtx, client, direct, obj.GetNamespace(), sensorHash, logger)
			if errConsistency != nil {
				return errors.Wrap(errConsistency, "failed to check TLS secrets CA consistency")
			}

			if consistent {
				break
			}

			time.Sleep(pollInterval)
		}

		logger.Info("All TLS secrets have new a CA, triggering rollout restart of workloads",
			"old-hash", storedHash,
			"new-hash", sensorHash)

		securedClusterServicesSelector := map[string]string{
			"app.kubernetes.io/part-of": "stackrox-secured-cluster-services",
		}
		if err := common.TriggerRolloutRestart(ctx, client, obj.GetNamespace(), securedClusterServicesSelector, logger); err != nil {
			return errors.Wrap(err, "failed to trigger rollout restart")
		}

		// Update the stored hash in the Secured Cluster custom resource
		patch := ctrlClient.MergeFrom(obj.DeepCopyObject().(ctrlClient.Object))
		annotations[sensorCAHashAnnotation] = sensorHash
		obj.SetAnnotations(annotations)
		if err := client.Patch(ctx, obj, patch); err != nil {
			return errors.Wrap(err, "failed to patch SecuredCluster with updated sensor CA hash")
		}
		logger.Info("Updated sensor CA hash in SecuredCluster annotations", "hash", sensorHash)

		return nil
	}
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
