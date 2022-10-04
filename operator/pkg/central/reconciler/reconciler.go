package reconciler

import (
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/central/extensions"
	"github.com/stackrox/rox/operator/pkg/central/values/translation"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/proxy"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	"github.com/stackrox/rox/operator/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	opts := []pkgReconciler.Option{
		pkgReconciler.WithExtraWatch(
			&source.Kind{Type: &platform.SecuredCluster{}},
			handleSiblingCentrals(mgr),
			// Only appearance and disappearance of a SecuredCluster resource can influence whether
			// an init bundle should be created by the Central controller.
			utils.CreateAndDeleteOnlyPredicate{},
		),
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralTLSExtensions(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralDBPasswordExtension(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcileScannerDBPasswordExtension(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcileAdminPasswordExtension(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), extensions.PVCTargetCentral, extensions.DefaultCentralPVCName)),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), extensions.PVCTargetCentralDB, extensions.DefaultCentralDBPVCName)),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
		pkgReconciler.WithReconcilePeriod(extensions.InitBundleReconcilePeriod),
		pkgReconciler.WithPauseReconcileAnnotation("stackrox.io/pause-reconcile"),
	}
	return reconciler.SetupReconcilerWithManager(
		mgr, platform.CentralGVK, image.CentralServicesChartPrefix,
		proxy.InjectProxyEnvVars(translation.Translator{}, proxyEnv),
		opts...,
	)
}
