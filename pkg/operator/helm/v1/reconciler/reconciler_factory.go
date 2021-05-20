package reconciler

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/charts"
	"github.com/stackrox/rox/pkg/operator-sdk/helm/controller"
	"github.com/stackrox/rox/pkg/operator-sdk/helm/release"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupReconciler creates a new helm reconciler and adds it to the controller manager.
func SetupReconciler(mgr ctrl.Manager, gvk schema.GroupVersionKind, chartPrefix string) error {
	chart, err := image.GetDefaultImage().LoadChart(chartPrefix, charts.RHACSMetaValues())
	if err != nil {
		return err
	}

	watchOptions := controller.WatchOptions{
		GVK:                     gvk,
		ManagerFactory:          release.NewManagerFactory(mgr, chart),
		WatchDependentResources: true,
		OverrideValues:          make(map[string]string),
	}

	if err := controller.Add(mgr, watchOptions); err != nil {
		return errors.Wrapf(err, "unable to add %s helm controller", gvk)
	}
	return nil
}
