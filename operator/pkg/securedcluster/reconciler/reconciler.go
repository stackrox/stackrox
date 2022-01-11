package reconciler

import (
	pkgReconciler "github.com/joelanford/helm-operator/pkg/reconciler"
	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/proxy"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	"github.com/stackrox/rox/operator/pkg/securedcluster/extensions"
	"github.com/stackrox/rox/operator/pkg/securedcluster/values/translation"
	"github.com/stackrox/rox/pkg/version"
	ctrl "sigs.k8s.io/controller-runtime"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	return reconciler.SetupReconcilerWithManager(mgr, platform.SecuredClusterGVK,
		image.SecuredClusterServicesChartPrefix,
		proxy.InjectProxyEnvVars(translation.NewTranslator(mgr.GetClient()), proxyEnv),
		pkgReconciler.WithPreExtension(extensions.CheckClusterNameExtension(nil)),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
	)
}
