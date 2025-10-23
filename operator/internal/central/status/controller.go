package status

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

// Reconciler reconciles deployment status and Helm reconciliation state into the Central CR status.
// This light-weight controller does not invoke Helm, it provides real-time updates for Ready and
// Progressing conditions.
type Reconciler struct {
	client.Client
}

// New creates a new Central status reconciler.
func New(c client.Client) *Reconciler {
	return &Reconciler{
		Client: c,
	}
}

// Reconcile reads deployment statuses and helm state, updates Ready and Progressing conditions.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Get the Central CR.
	central := &platform.Central{}
	if err := r.Get(ctx, req.NamespacedName, central); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if reconciliation is in progress.
	progressing, progressReason, progressMessage := r.determineProgressingState(central)

	// List all deployments owned by this Central.
	deployments := &appsv1.DeploymentList{}
	if err := r.List(ctx, deployments,
		client.InNamespace(central.Namespace),
		client.MatchingLabels{
			"app.kubernetes.io/instance":   central.Name,
			"app.kubernetes.io/managed-by": "stackrox-operator",
		},
	); err != nil {
		log.Error(err, "Failed to list deployments")
		return ctrl.Result{}, err
	}

	// Check if all deployments are ready
	ready, readyReason, readyMessage := r.determineReadyState(deployments.Items)

	// Update both conditions
	updatedConditions := central.Status.Conditions

	updatedConditions, progressingChanged := updateCondition(
		updatedConditions,
		"Progressing",
		progressing,
		progressReason,
		progressMessage,
	)

	updatedConditions, readyChanged := updateCondition(
		updatedConditions,
		"Ready",
		ready,
		readyReason,
		readyMessage,
	)
	anyChanged := progressingChanged || readyChanged

	if !anyChanged {
		// Nothing to update.
		return ctrl.Result{}, nil
	}

	central.Status.Conditions = updatedConditions

	// Update status subresource.
	// Note that in case of a conflict, we return the conflict here and the status reconciliation
	// will be automatically retried within controller-runtime with the latest version of the CR.
	if err := r.Status().Update(ctx, central); err != nil {
		log.Error(err, "Failed to update Central status")
		return ctrl.Result{}, err
	}

	log.Info("Updated status conditions",
		"progressing", progressing,
		"progressReason", progressReason,
		"ready", ready,
		"readyReason", readyReason,
	)
	return ctrl.Result{}, nil
}

// determineProgressingState infers if helm reconciliation is in progress.
func (r *Reconciler) determineProgressingState(central *platform.Central) (bool, platform.ConditionReason, string) {
	// Strategy 1: Check observedGeneration (most reliable)
	// If metadata.generation > status.observedGeneration, spec has changed and reconcile is pending
	if central.Generation > central.Status.ObservedGeneration {
		return true, "Reconciling", "Spec changes pending reconciliation"
	}

	// Strategy 2: Check helm conditions set by the operator
	for _, cond := range central.Status.Conditions {
		// If Deployed condition is Unknown, helm is working
		if cond.Type == platform.ConditionDeployed && cond.Status == platform.StatusUnknown {
			return true, "Reconciling", "Helm reconciliation in progress"
		}

		// If ReleaseFailed is True, reconciliation failed but might retry
		if cond.Type == platform.ConditionReleaseFailed && cond.Status == platform.StatusTrue {
			return true, "ReleaseFailed", cond.Message
		}

		// If Irreconcilable is True, there's a problem
		if cond.Type == platform.ConditionIrreconcilable && cond.Status == platform.StatusTrue {
			return true, "Irreconcilable", cond.Message
		}
	}

	// Strategy 3: Check if deployedRelease version matches reconciledVersion
	if central.Status.DeployedRelease != nil && central.Status.ReconciledVersion != "" {
		deployedVersion := central.Status.DeployedRelease.Version
		reconciledVersion := central.Status.ReconciledVersion

		// If versions don't match, reconciliation might be in progress
		if deployedVersion != reconciledVersion {
			return true, "VersionMismatch", "Deployed version differs from reconciled version"
		}
	}

	// No signs of active reconciliation
	return false, "ReconcileSuccessful", "Reconciliation completed successfully"
}

// determineReadyState checks if all deployments are ready.
func (r *Reconciler) determineReadyState(deployments []appsv1.Deployment) (bool, platform.ConditionReason, string) {
	if len(deployments) == 0 {
		return false, "NoDeployments", "No deployments found"
	}

	allReady := true
	notReadyCount := 0
	for _, dep := range deployments {
		if !isDeploymentReady(&dep) {
			allReady = false
			notReadyCount++
		}
	}

	if allReady {
		return true, "DeploymentsReady", "All deployments are ready"
	}

	return false, "DeploymentsNotReady",
		fmt.Sprintf("%d of %d deployments are not ready", notReadyCount, len(deployments))
}

// isDeploymentReady checks if a deployment has all replicas available.
func isDeploymentReady(dep *appsv1.Deployment) bool {
	for _, cond := range dep.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

// updateCondition updates or adds a condition in the conditions list.
// Returns the updated slice and whether anything changed.
func updateCondition(
	conditions []platform.StackRoxCondition,
	condType platform.ConditionType,
	status bool,
	reason platform.ConditionReason,
	message string,
) ([]platform.StackRoxCondition, bool) {
	condStatus := platform.StatusFalse
	if status {
		condStatus = platform.StatusTrue
	}

	// Find existing condition
	for i, cond := range conditions {
		if cond.Type == condType {
			// Check if update is needed
			if cond.Status == condStatus && cond.Reason == reason {
				return conditions, false // No change needed
			}

			// Update existing condition
			conditions[i].Status = condStatus
			conditions[i].Reason = reason
			conditions[i].Message = message
			conditions[i].LastTransitionTime = metav1.Time{Time: time.Now()}
			return conditions, true
		}
	}

	// Condition doesn't exist, create it
	newCondition := platform.StackRoxCondition{
		Type:               condType,
		Status:             condStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Time{Time: time.Now()},
	}

	return append(conditions, newCondition), true
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Watch Central CRs and also trigger reconciliation when deployments change
	return ctrl.NewControllerManagedBy(mgr).
		For(&platform.Central{}).
		Complete(r)
}
