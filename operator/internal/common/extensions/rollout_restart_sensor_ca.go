package extensions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	operatorCommon "github.com/stackrox/rox/operator/internal/common"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// SensorCAHashAnnotation stores the hash of the sensor CA to detect changes
	SensorCAHashAnnotation = "stackrox.io/sensor-ca-hash"
)

// RolloutRestartOnSensorCAChange is an extension that triggers rollout restarts of workloads
// when the sensor CA changes. This is triggered by the watcher on the tls-cert-sensor secret.
func RolloutRestartOnSensorCAChange(client ctrlClient.Client, logger logr.Logger) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), log logr.Logger) error {
		logger = logger.WithName("rollout-restart-sensor-ca")

		// Get the CA from the sensor TLS secret
		sensorCAHash, err := getSensorCAHash(ctx, client, obj.GetNamespace())
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				// Secret doesn't exist yet - this is normal for fresh installs
				logger.V(1).Info("sensor TLS secret not found, skipping rollout restart", "secretName", operatorCommon.SensorTLSSecretName)
				return nil
			}
			return errors.Wrap(err, "failed to get sensor CA hash")
		}

		// Get the stored hash from the custom resource annotations
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}

		storedHash := annotations[SensorCAHashAnnotation]
		if storedHash == "" {
			// No stored hash yet - this is a fresh install or first time seeing the secret
			// Store the hash but don't trigger a restart
			logger.V(1).Info("No stored sensor CA hash found, storing current hash without triggering restart")
			annotations[SensorCAHashAnnotation] = sensorCAHash
			obj.SetAnnotations(annotations)
			return nil
		}

		if storedHash == sensorCAHash {
			// Hash hasn't changed, no need to restart
			logger.V(1).Info("Sensor CA hash unchanged, skipping rollout restart")
			return nil
		}

		// Hash has changed, indicating sensor got new CA from Central during cert refresh
		logger.Info("Sensor CA changed, triggering rollout restart of workloads",
			"old-hash", storedHash,
			"new-hash", sensorCAHash)

		if err := TriggerRolloutRestart(ctx, client, obj.GetNamespace(), logger); err != nil {
			return errors.Wrap(err, "failed to trigger rollout restart")
		}

		// Update the stored hash in the custom resource
		annotations[SensorCAHashAnnotation] = sensorCAHash
		obj.SetAnnotations(annotations)

		return nil
	}
}

// getSensorCAHash retrieves the sensor TLS secret and returns a hash of its ca.pem content
func getSensorCAHash(ctx context.Context, client ctrlClient.Client, namespace string) (string, error) {
	var secret corev1.Secret
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      operatorCommon.SensorTLSSecretName,
	}

	if err := client.Get(ctx, key, &secret); err != nil {
		return "", err
	}

	caPEM := secret.Data["ca.pem"]
	if len(caPEM) == 0 {
		return "", errors.Errorf("ca.pem is empty in %s secret", operatorCommon.SensorTLSSecretName)
	}

	// Simple hash of the CA content - this changes when sensor gets new CA from Central
	return fmt.Sprintf("%x", hash(string(caPEM))), nil
}
