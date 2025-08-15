package common

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
	// restartedAtAnnotation is an annotation added to the workload's pod template, used to trigger pod restarts
	// (similar to kubectl rollout restart)
	restartedAtAnnotation = "app.stackrox.io/restartedAt"
)

// TriggerRolloutRestart adds the restartedAt annotation to all Deployments and DaemonSets
// matching the given label selector, causing Kubernetes to restart their pods
func TriggerRolloutRestart(ctx context.Context, client ctrlClient.Client, namespace string, labelSelector map[string]string, logger logr.Logger) error {
	restartTime := time.Now().Format(time.RFC3339)

	// List all Deployments matching the label selector
	var deployments appsv1.DeploymentList
	listOpts := []ctrlClient.ListOption{ctrlClient.InNamespace(namespace)}
	if len(labelSelector) > 0 {
		listOpts = append(listOpts, ctrlClient.MatchingLabels(labelSelector))
	}
	if err := client.List(ctx, &deployments, listOpts...); err != nil {
		return errors.Wrap(err, "failed to list deployments")
	}

	// Restart all Deployments
	for i := range deployments.Items {
		deployment := &deployments.Items[i]
		if err := addRestartAnnotation(ctx, client, deployment, restartTime, logger); err != nil {
			logger.Error(err, "failed to restart deployment", "deployment", deployment.Name)
		}
	}

	// List all DaemonSets matching the label selector
	var daemonSets appsv1.DaemonSetList
	if err := client.List(ctx, &daemonSets, listOpts...); err != nil {
		return errors.Wrap(err, "failed to list daemonsets")
	}

	// Restart all DaemonSets
	for i := range daemonSets.Items {
		daemonSet := &daemonSets.Items[i]
		if err := addRestartAnnotation(ctx, client, daemonSet, restartTime, logger); err != nil {
			logger.Error(err, "failed to restart daemonset", "daemonset", daemonSet.Name)
		}
	}

	logger.Info("Triggered rollout restart for all workloads", "restart-time", restartTime)
	return nil
}

// addRestartAnnotation adds the restartedAt annotation to trigger a rollout restart
func addRestartAnnotation(ctx context.Context, client ctrlClient.Client, obj ctrlClient.Object, restartTime string, logger logr.Logger) error {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	original := obj.DeepCopyObject().(ctrlClient.Object)

	// Add the restart annotation to the pod template
	switch workload := obj.(type) {
	case *appsv1.Deployment:
		if workload.Spec.Template.Annotations == nil {
			workload.Spec.Template.Annotations = make(map[string]string)
		}
		workload.Spec.Template.Annotations[restartedAtAnnotation] = restartTime

	case *appsv1.DaemonSet:
		if workload.Spec.Template.Annotations == nil {
			workload.Spec.Template.Annotations = make(map[string]string)
		}
		workload.Spec.Template.Annotations[restartedAtAnnotation] = restartTime

	default:
		return errors.Errorf("unsupported workload type: %T", obj)
	}

	patch := ctrlClient.MergeFrom(original)
	if err := client.Patch(ctx, obj, patch); err != nil {
		return errors.Wrapf(err, "failed to patch %s %s", obj.GetObjectKind().GroupVersionKind().Kind, key)
	}

	logger.Info("Added restart annotation to workload",
		"kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"name", obj.GetName(),
		"restart-time", restartTime)

	return nil
}
