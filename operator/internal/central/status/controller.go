package status

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
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

	return nil
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
