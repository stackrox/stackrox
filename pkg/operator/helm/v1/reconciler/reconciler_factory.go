package reconciler

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/charts"
	"github.com/stackrox/rox/pkg/operator-sdk/helm/controller"
	"github.com/stackrox/rox/pkg/operator-sdk/helm/release"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// createWatchOptions creates the watch options for helm operator v1
func createWatchOptions(mgr manager.Manager, gvk schema.GroupVersionKind, chartPrefix string) (controller.WatchOptions, error) {
	templateImage := image.GetDefaultImage()
	renderedChartFiles, err := templateImage.LoadAndInstantiateChartTemplate(chartPrefix, charts.RHACSMetaValues())
	if err != nil {
		return controller.WatchOptions{}, errors.Wrapf(err, "loading and instantiating embedded chart %q failed", chartPrefix)
	}

	chart, err := loader.LoadFiles(renderedChartFiles)
	if err != nil {
		return controller.WatchOptions{}, errors.Wrapf(err, "loading %q helm chart files failed", chartPrefix)
	}
	return controller.WatchOptions{
		GVK:                     gvk,
		ManagerFactory:          release.NewManagerFactory(mgr, chart),
		WatchDependentResources: true,
		OverrideValues:          make(map[string]string),
	}, nil
}

// SetupReconciler creates a new helm reconciler and adds it to the controller manager.
func SetupReconciler(mgr ctrl.Manager, gvk schema.GroupVersionKind, chartPrefix string) error {
	watchOptions, err := createWatchOptions(mgr, gvk, chartPrefix)
	if err != nil {
		return errors.Wrapf(err, "unable to create WatchOptions for %s controller", gvk)
	}

	if err := controller.Add(mgr, watchOptions); err != nil {
		return errors.Wrapf(err, "unable to add %s helm controller", gvk)
	}
	return nil
}
