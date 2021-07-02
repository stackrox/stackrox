package reconciler

import (
	pkgReconciler "github.com/joelanford/helm-operator/pkg/reconciler"
	"github.com/stackrox/rox/image"
	securedClusterv1Alpha1 "github.com/stackrox/rox/operator/api/securedcluster/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	"github.com/stackrox/rox/operator/pkg/securedcluster/extensions"
	"github.com/stackrox/rox/operator/pkg/securedcluster/values/translation"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager, client kubernetes.Interface) error {
	return reconciler.SetupReconcilerWithManager(mgr, securedClusterv1Alpha1.SecuredClusterGVK,
		image.SecuredClusterServicesChartPrefix,
		translation.NewTranslator(client),
		pkgReconciler.WithPreExtension(extensions.CheckClusterNameExtension(client)))
}
