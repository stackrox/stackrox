package reconciler

import (
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/operator/api/central/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/central/values/translation"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

const centralKind = "Central"

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager) error {
	gvk := schema.GroupVersionKind{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Kind: centralKind}
	return reconciler.SetupReconcilerWithManager(mgr, gvk, image.CentralServicesChartPrefix, translation.Translator{Config: mgr.GetConfig()})
}
