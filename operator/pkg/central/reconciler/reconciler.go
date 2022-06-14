package reconciler

import (
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/stackrox/stackrox/image"
	platform "github.com/stackrox/stackrox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/stackrox/operator/pkg/central/extensions"
	"github.com/stackrox/stackrox/operator/pkg/central/values/translation"
	commonExtensions "github.com/stackrox/stackrox/operator/pkg/common/extensions"
	"github.com/stackrox/stackrox/operator/pkg/proxy"
	"github.com/stackrox/stackrox/operator/pkg/reconciler"
	"github.com/stackrox/stackrox/operator/pkg/utils"
	"github.com/stackrox/stackrox/pkg/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	return reconciler.SetupReconcilerWithManager(
		mgr, platform.CentralGVK, image.CentralServicesChartPrefix,
		proxy.InjectProxyEnvVars(translation.Translator{}, proxyEnv),
		pkgReconciler.WithExtraWatch(
			&source.Kind{Type: &platform.SecuredCluster{}},
			handleSiblingCentrals(mgr),
			// Only appearance and disappearance of a SecuredCluster resource can influence whether
			// an init bundle should be created by the Central controller.
			utils.CreateAndDeleteOnlyPredicate{}),
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralTLSExtensions(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcileScannerDBPasswordExtension(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcileAdminPasswordExtension(mgr.GetClient())),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient())),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
		pkgReconciler.WithReconcilePeriod(extensions.InitBundleReconcilePeriod),
	)
}
