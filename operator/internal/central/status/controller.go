package status

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// Reconciler reconciles deployment status and Helm reconciliation state into the Central CR status.
// This light-weight controller does not invoke Helm, it provides real-time updates for Available and
// Progressing conditions.
type Reconciler struct {
	ctrlClient.Client
}

// New creates a new Central status reconciler.
func New(c ctrlClient.Client) *Reconciler {
	return &Reconciler{
		Client: c,
	}
}

// Reconcile reads deployment statuses and helm state, updates Available and Progressing conditions.
// It implements a retry mechanism for conflict errors using the standard Kubernetes retry utilities.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Central status controller reconciliation started")

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.runReconciliationFlow(ctx, log, req)
	})

	return ctrl.Result{}, err
}

func (r *Reconciler) runReconciliationFlow(ctx context.Context, log logr.Logger, req ctrl.Request) error {
	// Get the Central CR.
	central := &platform.Central{}
	if err := r.Get(ctx, req.NamespacedName, central); err != nil {
		return ctrlClient.IgnoreNotFound(err)
	}

	// Update condition "Progressing".
	progressingChanged := r.updateProgressing(ctx, central)
	if progressingChanged {
		progCond := getCondition(central.Status.Conditions, "Progressing")
		if progCond != nil {
			log.Info("Progressing condition updated", "status", progCond.Status, "reason", progCond.Reason)
		}
	}

	// Update status subresource.
	// Conflicts are handled by the retry mechanism in the Reconcile function.
	log.Info("Updating Central status")
	if err := r.Status().Update(ctx, central); err != nil {
		if !errors.IsConflict(err) {
			log.Error(err, "Failed to update Central status")
		}
		return err
	}

	log.Info("Central status updated successfully")
	return nil
}

// updateProgressing updates the Progressing condition based on helm reconciliation state.
// Returns true if the condition changed.
func (r *Reconciler) updateProgressing(_ context.Context, central *platform.Central) bool {
	progressing, reason, message := r.determineProgressingState(central)

	var changed bool
	central.Status.Conditions, changed = updateCondition(
		central.Status.Conditions,
		"Progressing",
		progressing,
		reason,
		message,
	)

	return changed
}

// determineProgressingState infers if Helm reconciliation is in progress.
// Returns (isProgressing, reason, message).
func (r *Reconciler) determineProgressingState(central *platform.Central) (bool, platform.ConditionReason, string) {
	// Check observedGeneration.
	// If metadata.generation > status.observedGeneration, spec has changed and reconcile is pending
	if central.Generation > central.Status.ObservedGeneration {
		return true, "Reconciling", "Spec changes pending reconciliation"
	}

	// Check Helm conditions set by the operator.
	for _, cond := range central.Status.Conditions {
		// If Deployed condition is Unknown, helm is working
		if cond.Type == platform.ConditionDeployed && cond.Status != platform.StatusTrue {
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

	// No signs of active reconciliation.
	return false, "ReconcileSuccessful", "Reconciliation completed successfully"
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
	// Create controller using low-level API to avoid extra logging fields
	c, err := controller.New("central-status-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	err = c.Watch(
		source.Kind(mgr.GetCache(), &platform.Central{},
			&handler.TypedEnqueueRequestForObject[*platform.Central]{},
		),
	)
	if err != nil {
		return err
	}

	// Watch owned Deployments to react to deployment status changes
	err = c.Watch(
		source.Kind(mgr.GetCache(), &appsv1.Deployment{},
			handler.TypedEnqueueRequestForOwner[*appsv1.Deployment](
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&platform.Central{},
				handler.OnlyControllerOwner(),
			),
		),
	)
	if err != nil {
		return err
	}

	return nil
}
