package status

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Reconciler reconciles deployment status and Helm reconciliation state in the CR status.
// This light-weight controller does not invoke Helm, it provides real-time updates for Available and
// Progressing conditions.
type Reconciler[T platform.ObjectForStatusController] struct {
	ctrlClient.Client
	name string
}

// New creates a new status reconciler.
func New[T platform.ObjectForStatusController](c ctrlClient.Client, name string) *Reconciler[T] {
	name = fmt.Sprintf("%s-status-controller", strings.ToLower(name))
	return &Reconciler[T]{
		Client: c,
		name:   name,
	}
}

// Reconcile reads deployment statuses and helm state, updates Available and Progressing conditions.
// It implements a retry mechanism for conflict errors using the standard Kubernetes retry utilities.
func (r *Reconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log = log.WithName(r.name)
	log.Info("Status reconciliation initiated")

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.runReconciliationFlow(ctx, log, req)
	})

	return ctrl.Result{}, err
}

func (r *Reconciler[T]) runReconciliationFlow(ctx context.Context, log logr.Logger, req ctrl.Request) error {
	// Create a new instance of T using reflection
	typeOfT := reflect.TypeOf(new(T)).Elem()
	typeOfDerefT := typeOfT.Elem()
	obj := reflect.New(typeOfDerefT).Interface().(T)

	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrlClient.IgnoreNotFound(err)
	}

	anyChanges := false

	// Update condition "Progressing".
	updatedProgressingCond := r.updateProgressing(ctx, obj)
	if updatedProgressingCond != nil {
		log.Info("Progressing condition updated",
			"status", updatedProgressingCond.Status,
			"reason", updatedProgressingCond.Reason)
		anyChanges = true
	}

	// Update condition "Available".
	updatedAvailableCond := r.updateAvailable(ctx, obj)
	if updatedAvailableCond != nil {
		log.Info("Available condition updated",
			"status", updatedAvailableCond.Status,
			"reason", updatedAvailableCond.Reason)
		anyChanges = true
	}

	// If nothing changed, skip the status update.
	if !anyChanges {
		log.V(1).Info("No status changes detected, skipping update")
		return nil
	}

	// Update status subresource.
	// Conflicts are handled by the retry mechanism in the Reconcile function.
	log.Info("Updating status")
	if err := r.Status().Update(ctx, obj); err != nil {
		if !errors.IsConflict(err) {
			log.Error(err, "Failed to update status")
		}
		return err
	}

	log.Info("Status updated")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler[T]) SetupWithManager(mgr ctrl.Manager) error {
	// Create controller using low-level API to avoid extra logging fields
	c, err := controller.New(r.name, mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	// Watch CRs with a predicate to filter away status-only updates
	typeOfT := reflect.TypeOf(new(T)).Elem()
	typeOfDerefT := typeOfT.Elem()
	emptyCR := reflect.New(typeOfDerefT).Interface().(T)

	err = c.Watch(
		source.Kind(mgr.GetCache(), emptyCR,
			&handler.TypedEnqueueRequestForObject[T]{},
			SkipStatusControllerUpdates[T]{},
		),
	)
	if err != nil {
		return err
	}

	// Watch owned Deployments to react to deployment status changes
	emptyCR = reflect.New(typeOfDerefT).Interface().(T)
	err = c.Watch(
		source.Kind(mgr.GetCache(), &appsv1.Deployment{},
			handler.TypedEnqueueRequestForOwner[*appsv1.Deployment](
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				emptyCR,
				handler.OnlyControllerOwner(),
			),
		),
	)
	if err != nil {
		return err
	}

	return nil
}

// updateProgressing updates the Progressing condition based on helm reconciliation state.
// Returns nil if the condition is unchanged, otherwise returns the new condition.
func (r *Reconciler[T]) updateProgressing(_ context.Context, obj T) *platform.StackRoxCondition {
	prorgressingStatus, reason, message := r.determineProgressingState(obj)

	newCond := platform.StackRoxCondition{
		Type:               platform.ConditionProgressing,
		Status:             prorgressingStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Time{Time: time.Now()},
	}

	condChanged := obj.SetCondition(newCond)
	if condChanged {
		return &newCond
	}
	return nil
}

// determineProgressingState infers if Helm reconciliation is in progress.
// Returns (isProgressing, reason, message).
func (r *Reconciler[T]) determineProgressingState(obj T) (platform.ConditionStatus, platform.ConditionReason, string) {
	// Check observedGeneration. If metadata.generation > status.observedGeneration, spec has changed and reconcile
	// is pending.
	if obj.GetGeneration() > obj.GetObservedGeneration() {
		return platform.StatusTrue, "Reconciling", "Reconciliation in progress"
	}

	// observedGeneration matches generation.

	// If Irreconcilable is True, surface it.
	if cond := obj.GetCondition(platform.ConditionIrreconcilable); cond != nil && cond.Status == platform.StatusTrue {
		return platform.StatusTrue, "Irreconcilable", cond.Message
	}

	return platform.StatusFalse, "ReconcileSuccessful", "Reconciliation completed"
}

// updateAvailable updates the Available condition based on deployment readiness.
// Returns true if the condition changed.
func (r *Reconciler[T]) updateAvailable(ctx context.Context, obj T) *platform.StackRoxCondition {
	log := log.FromContext(ctx)

	// List all deployments owned by the resource currently under reconciliation.
	deployments := &appsv1.DeploymentList{}
	err := r.List(ctx, deployments,
		ctrlClient.InNamespace(obj.GetNamespace()),
		ctrlClient.MatchingLabels{
			"app.kubernetes.io/instance": obj.GetName(),
			"app.stackrox.io/managed-by": "operator",
		},
	)
	if err != nil {
		log.Error(err, "Failed to list deployments")
		return nil
	}

	availableStatus, reason, message := determineAvailableState(deployments.Items)

	newCond := platform.StackRoxCondition{
		Type:               platform.ConditionAvailable,
		Status:             availableStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Time{Time: time.Now()},
	}

	condChanged := obj.SetCondition(newCond)
	if condChanged {
		return &newCond
	}
	return nil
}

// determineAvailableState checks if all deployments are available.
func determineAvailableState(deployments []appsv1.Deployment) (platform.ConditionStatus, platform.ConditionReason, string) {
	if len(deployments) == 0 {
		return platform.StatusFalse, "NoDeployments", "No deployments found"
	}

	var notReadyNames []string
	for _, dep := range deployments {
		if !isDeploymentReady(&dep) {
			notReadyNames = append(notReadyNames, dep.Name)
		}
	}

	if len(notReadyNames) == 0 {
		return platform.StatusTrue, "DeploymentsReady", "All deployments are ready"
	}

	// Sort to avoid updates merely due to ordering changes.
	slices.Sort(notReadyNames)

	return platform.StatusFalse, "DeploymentsNotReady",
		fmt.Sprintf("%d of %d deployments are not ready: %s", len(notReadyNames), len(deployments), strings.Join(notReadyNames, ", "))
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
