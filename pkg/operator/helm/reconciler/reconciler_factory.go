package reconciler

import (
	"github.com/joelanford/helm-operator/pkg/reconciler"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/charts"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupReconcilerWithManager creates and registers a new helm reconciler to the given controller manager.
func SetupReconcilerWithManager(mgr ctrl.Manager, gvk schema.GroupVersionKind, chartPrefix string) error {
	chart, err := image.GetDefaultImage().LoadChart(chartPrefix, charts.RHACSMetaValues())
	if err != nil {
		return err
	}

	reconciler, err := reconciler.New(
		reconciler.WithChart(*chart),
		reconciler.WithGroupVersionKind(gvk),
	)
	if err != nil {
		return errors.Wrapf(err, "unable to create %s reconciler", gvk)
	}

	if err := reconciler.SetupWithManager(mgr); err != nil {
		return errors.Wrapf(err, "unable to setup %s reconciler", gvk)
	}
	return nil
}
