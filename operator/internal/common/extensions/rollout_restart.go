package extensions

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// RestartedAtAnnotation is the annotation used to trigger pod restarts (similar to kubectl rollout restart)
	RestartedAtAnnotation = "deployment.kubernetes.io/restartedAt"
)

// TriggerRolloutRestart adds the restartedAt annotation to all Deployments and DaemonSets
// managed by the operator, causing Kubernetes to restart their pods
func TriggerRolloutRestart(ctx context.Context, client ctrlClient.Client, namespace string, logger logr.Logger) error {
	restartTime := time.Now().Format(time.RFC3339)

	// List all Deployments managed by the operator
	var deployments appsv1.DeploymentList
	if err := client.List(ctx, &deployments,
		ctrlClient.InNamespace(namespace),
	); err != nil {
		return errors.Wrap(err, "failed to list deployments")
	}

	// Restart all Deployments
	for i := range deployments.Items {
		deployment := &deployments.Items[i]
		if err := addRestartAnnotation(ctx, client, deployment, restartTime, logger); err != nil {
			logger.Error(err, "failed to restart deployment", "deployment", deployment.Name)
			// Continue with other deployments even if one fails
		}
	}

	// List all DaemonSets managed by the operator
	var daemonSets appsv1.DaemonSetList
	if err := client.List(ctx, &daemonSets,
		ctrlClient.InNamespace(namespace),
	); err != nil {
		return errors.Wrap(err, "failed to list daemonsets")
	}

	// Restart all DaemonSets
	for i := range daemonSets.Items {
		daemonSet := &daemonSets.Items[i]
		if err := addRestartAnnotation(ctx, client, daemonSet, restartTime, logger); err != nil {
			logger.Error(err, "failed to restart daemonset", "daemonset", daemonSet.Name)
			// Continue with other daemonsets even if one fails
		}
	}

	logger.Info("Triggered rollout restart for all workloads", "restart-time", restartTime)
	return nil
}

// addRestartAnnotation adds the restartedAt annotation to trigger a rollout restart
func addRestartAnnotation(ctx context.Context, client ctrlClient.Client, obj ctrlClient.Object, restartTime string, logger logr.Logger) error {
	// Get the current object to ensure we have the latest version
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	// Update via patch to avoid conflicts
	patch := ctrlClient.MergeFrom(obj.DeepCopyObject().(ctrlClient.Object))

	// Add the restart annotation to the pod template
	switch workload := obj.(type) {
	case *appsv1.Deployment:
		if workload.Spec.Template.Annotations == nil {
			workload.Spec.Template.Annotations = make(map[string]string)
		}
		workload.Spec.Template.Annotations[RestartedAtAnnotation] = restartTime

	case *appsv1.DaemonSet:
		if workload.Spec.Template.Annotations == nil {
			workload.Spec.Template.Annotations = make(map[string]string)
		}
		workload.Spec.Template.Annotations[RestartedAtAnnotation] = restartTime

	default:
		return errors.Errorf("unsupported workload type: %T", obj)
	}

	if err := client.Patch(ctx, obj, patch); err != nil {
		return errors.Wrapf(err, "failed to patch %s %s", obj.GetObjectKind().GroupVersionKind().Kind, key)
	}

	logger.Info("Added restart annotation to workload",
		"kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"name", obj.GetName(),
		"restart-time", restartTime)

	return nil
}

// hash is a simple hash function
func hash(s string) int {
	h := 0
	for _, c := range s {
		h = 31*h + int(c)
	}
	return h
}
