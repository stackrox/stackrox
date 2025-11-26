package status

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

// Reconciler reconciles deployment status and Helm reconciliation state in the CR status.
// This light-weight controller does not invoke Helm, it provides real-time updates for Available and
// Progressing conditions.
type Reconciler[T platform.ObjectForStatusController] struct {
	ctrlClient.Client
	name          string
	lowercaseName string
}

// New creates a new status reconciler.
func New[T platform.ObjectForStatusController](c ctrlClient.Client, name string) *Reconciler[T] {
	return &Reconciler[T]{
		Client:        c,
		name:          name,
		lowercaseName: strings.ToLower(name),
	}
}

// Reconcile reads deployment statuses and helm state, updates Available and Progressing conditions.
// It implements a retry mechanism for conflict errors using the standard Kubernetes retry utilities.
func (r *Reconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log = log.WithName("FOO")
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

	// Update condition "Progressing".
	progressingChanged := r.updateProgressing(ctx, obj)
	if progressingChanged {
		progCond := obj.GetCondition(platform.ConditionProgressing)
		if progCond != nil {
			log.Info("Progressing condition updated", "status", progCond.Status, "reason", progCond.Reason)
		}
	}

	// If nothing changed, skip the status update.
	if !progressingChanged {
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
	controllerName := fmt.Sprintf("%s-status-controller", r.lowercaseName)
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	// Watch CRs with predicate to filter status-only updates
	typeOfT := reflect.TypeOf(new(T)).Elem()
	typeOfDerefT := typeOfT.Elem()
	emptyCR := reflect.New(typeOfDerefT).Interface().(T)

	err = c.Watch(
		source.Kind(mgr.GetCache(), emptyCR,
			&handler.TypedEnqueueRequestForObject[T]{},
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
// Returns true if the condition changed.
func (r *Reconciler[T]) updateProgressing(_ context.Context, obj T) bool {
	prorgressingStatus, reason, message := r.determineProgressingState(obj)

	return obj.SetCondition(platform.StackRoxCondition{
		Type:    platform.ConditionProgressing,
		Status:  prorgressingStatus,
		Reason:  reason,
		Message: message,
		// LastTransitionTime: metav1.Time{Time: time.Now()},
	})
}

// determineProgressingState infers if Helm reconciliation is in progress.
// Returns (isProgressing, reason, message).
func (r *Reconciler[T]) determineProgressingState(obj T) (platform.ConditionStatus, platform.ConditionReason, string) {
	// Check observedGeneration.
	// If metadata.generation > status.observedGeneration, spec has changed and reconcile is pending
	if obj.GetGeneration() > obj.GetObservedGeneration() {
		return platform.StatusTrue, "Reconciling", "Spec changes pending reconciliation"
	}

	// If Deployed condition is Unknown, helm is working
	cond := obj.GetCondition(platform.ConditionDeployed)
	if cond.Type == platform.ConditionDeployed && cond.Status != platform.StatusTrue {
		return platform.StatusTrue, "Reconciling", "Helm reconciliation in progress"
	}

	// If ReleaseFailed is True, reconciliation failed but might retry
	cond = obj.GetCondition(platform.ConditionReleaseFailed)
	if cond.Type == platform.ConditionReleaseFailed && cond.Status == platform.StatusTrue {
		return platform.StatusTrue, "ReleaseFailed", cond.Message
	}

	// If Irreconcilable is True, there's a problem
	cond = obj.GetCondition(platform.ConditionIrreconcilable)
	if cond.Type == platform.ConditionIrreconcilable && cond.Status == platform.StatusTrue {
		return platform.StatusTrue, "Irreconcilable", cond.Message
	}

	// No signs of active reconciliation.
	return platform.StatusFalse, "ReconcileSuccessful", "Reconciliation completed"
}
