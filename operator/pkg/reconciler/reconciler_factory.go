package reconciler

import (
	"github.com/joelanford/helm-operator/pkg/reconciler"
	"github.com/joelanford/helm-operator/pkg/values"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/charts"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupReconcilerWithManager creates and registers a new helm reconciler to the given controller manager.
func SetupReconcilerWithManager(mgr ctrl.Manager, gvk schema.GroupVersionKind, chartPrefix string, translator values.Translator, extraOpts ...reconciler.Option) error {
	chart, err := image.GetDefaultImage().LoadChart(chartPrefix, charts.RHACSMetaValues())
	if err != nil {
		return err
	}

	reconcilerOpts := []reconciler.Option{
		reconciler.WithChart(*chart),
		reconciler.WithGroupVersionKind(gvk),
		reconciler.WithValueTranslator(translator),
		//TODO(ROX-7362): re-evaluate enabling depended watches
		reconciler.SkipDependentWatches(true),
	}
	reconcilerOpts = append(reconcilerOpts, extraOpts...)

	r, err := reconciler.New(reconcilerOpts...)
	if err != nil {
		return errors.Wrapf(err, "unable to create %s reconciler", gvk)
	}

	if err := r.SetupWithManager(mgr, reconciler.SetupOpts{DisableSetupScheme: true}); err != nil {
		return errors.Wrapf(err, "unable to setup %s reconciler", gvk)
	}
	return nil
}
