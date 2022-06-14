package reconciler

import (
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/stackrox/stackrox/image"
	platform "github.com/stackrox/stackrox/operator/apis/platform/v1alpha1"
	commonExtensions "github.com/stackrox/stackrox/operator/pkg/common/extensions"
	"github.com/stackrox/stackrox/operator/pkg/proxy"
	"github.com/stackrox/stackrox/operator/pkg/reconciler"
	"github.com/stackrox/stackrox/operator/pkg/securedcluster/extensions"
	"github.com/stackrox/stackrox/operator/pkg/securedcluster/values/translation"
	"github.com/stackrox/stackrox/operator/pkg/utils"
	"github.com/stackrox/stackrox/pkg/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	return reconciler.SetupReconcilerWithManager(
		mgr, platform.SecuredClusterGVK,
		image.SecuredClusterServicesChartPrefix,
		proxy.InjectProxyEnvVars(translation.NewTranslator(mgr.GetClient()), proxyEnv),
		pkgReconciler.WithExtraWatch(
			&source.Kind{Type: &platform.Central{}},
			handleSiblingSecuredClusters(mgr),
			// Only appearance and disappearance of a Central resource can influence whether
			// a local scanner should be deployed by the SecuredCluster controller.
			utils.CreateAndDeleteOnlyPredicate{}),
		pkgReconciler.WithPreExtension(extensions.CheckClusterNameExtension(nil)),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
		pkgReconciler.WithPreExtension(extensions.ReconcileLocalScannerDBPasswordExtension(mgr.GetClient())),
	)
}
