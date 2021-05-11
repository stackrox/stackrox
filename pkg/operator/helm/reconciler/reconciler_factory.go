package reconciler

import (
	"github.com/joelanford/helm-operator/pkg/reconciler"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/charts"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// create creates and configures the reconciler
func create(gvk schema.GroupVersionKind, chartPrefix string) (*reconciler.Reconciler, error) {
	templateImage := image.GetDefaultImage()
	renderedChartFiles, err := templateImage.LoadAndInstantiateChartTemplate(chartPrefix, charts.RHACSMetaValues())
	if err != nil {
		return nil, errors.Wrapf(err, "loading and instantiating embedded chart %q failed", chartPrefix)
	}

	chart, err := loader.LoadFiles(renderedChartFiles)
	if err != nil {
		return nil, errors.Wrapf(err, "loading %q helm chart files failed", chartPrefix)
	}

	return reconciler.New(
		reconciler.WithChart(*chart),
		reconciler.WithGroupVersionKind(gvk),
	)
}

// SetupReconcilerWithManager creates and registers a new helm reconciler to the given controller manager.
func SetupReconcilerWithManager(mgr ctrl.Manager, gvk schema.GroupVersionKind, chartPrefix string) error {
	reconciler, err := create(gvk, chartPrefix)
	if err != nil {
		return errors.Wrapf(err, "unable to create %s reconciler", gvk)
	}

	if err := reconciler.SetupWithManager(mgr); err != nil {
		return errors.Wrapf(err, "unable to create %s reconciler", gvk)
	}
	return nil
}
