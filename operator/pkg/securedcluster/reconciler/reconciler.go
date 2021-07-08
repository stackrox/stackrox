package reconciler

import (
	pkgReconciler "github.com/joelanford/helm-operator/pkg/reconciler"
	"github.com/stackrox/rox/image"
	securedClusterv1Alpha1 "github.com/stackrox/rox/operator/api/securedcluster/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/proxy"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	"github.com/stackrox/rox/operator/pkg/securedcluster/extensions"
	"github.com/stackrox/rox/operator/pkg/securedcluster/values/translation"
	"github.com/stackrox/rox/pkg/version"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager, client kubernetes.Interface) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	return reconciler.SetupReconcilerWithManager(mgr, securedClusterv1Alpha1.SecuredClusterGVK,
		image.SecuredClusterServicesChartPrefix,
		proxy.InjectProxyEnvVars(translation.NewTranslator(client), proxyEnv),
		pkgReconciler.WithPreExtension(extensions.CheckClusterNameExtension(client)),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(client, proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
	)
}
