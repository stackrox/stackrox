package reconciler

import (
	pkgReconciler "github.com/joelanford/helm-operator/pkg/reconciler"
	"github.com/stackrox/rox/image"
	centralV1Alpha1 "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/central/extensions"
	"github.com/stackrox/rox/operator/pkg/central/values/translation"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/proxy"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	"github.com/stackrox/rox/pkg/version"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager, client kubernetes.Interface) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	return reconciler.SetupReconcilerWithManager(
		mgr, centralV1Alpha1.CentralGVK, image.CentralServicesChartPrefix,
		proxy.InjectProxyEnvVars(translation.Translator{Client: client}, proxyEnv),
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralTLSExtensions(client)),
		pkgReconciler.WithPreExtension(extensions.ReconcileScannerDBPasswordExtension(client)),
		pkgReconciler.WithPreExtension(extensions.ReconcileAdminPasswordExtension(client)),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(client)),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(client, proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
	)
}
