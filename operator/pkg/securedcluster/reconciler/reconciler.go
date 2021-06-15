package reconciler

import (
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/operator/api/securedcluster/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	"github.com/stackrox/rox/operator/pkg/securedcluster/values/translation"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

const securedClusterKind = "SecuredCluster"

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager) error {
	securedClusterGVK := schema.GroupVersionKind{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Kind: securedClusterKind}
	return reconciler.SetupReconcilerWithManager(mgr, securedClusterGVK, image.SecuredClusterServicesChartPrefix,
		translation.NewTranslator(mgr.GetConfig()))
}
