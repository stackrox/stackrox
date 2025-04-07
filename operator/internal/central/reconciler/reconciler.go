package reconciler

import (
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/central/extensions"
	centralTranslation "github.com/stackrox/rox/operator/internal/central/values/translation"
	commonExtensions "github.com/stackrox/rox/operator/internal/common/extensions"
	"github.com/stackrox/rox/operator/internal/legacy"
	"github.com/stackrox/rox/operator/internal/proxy"
	"github.com/stackrox/rox/operator/internal/reconciler"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/operator/internal/values/translation"
	"github.com/stackrox/rox/pkg/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager, selector string) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	opts := []pkgReconciler.Option{
		pkgReconciler.WithExtraWatch(
			source.Kind[*platform.SecuredCluster](
				mgr.GetCache(),
				&platform.SecuredCluster{},
				reconciler.HandleSiblings[*platform.SecuredCluster](platform.CentralGVK, mgr),
				// Only appearance and disappearance of a SecuredCluster resource can influence whether
				// an init bundle should be created by the Central controller.
				utils.CreateAndDeleteOnlyPredicate[*platform.SecuredCluster]{})),
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralTLSExtensions(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralTLSExtensions(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralDBPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileScannerDBPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileScannerV4DBPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileAdminPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), mgr.GetAPIReader(), extensions.PVCTargetCentral, extensions.DefaultCentralPVCName)),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), mgr.GetAPIReader(), extensions.PVCTargetCentralDB, extensions.DefaultCentralDBPVCName)),
		pkgReconciler.WithPreExtension(extensions.ReconcileScannerV4StatusDefaultsExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), mgr.GetAPIReader(), proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
		pkgReconciler.WithReconcilePeriod(extensions.InitBundleReconcilePeriod),
		pkgReconciler.WithPauseReconcileAnnotation(commonExtensions.PauseReconcileAnnotation),
	}

	opts, err := commonExtensions.AddSelectorOptionIfNeeded(selector, opts)
	if err != nil {
		return err
	}

	opts = commonExtensions.AddMapKubeAPIsExtensionIfMapFileExists(opts)

	return reconciler.SetupReconcilerWithManager(
		mgr, platform.CentralGVK, image.CentralServicesChartPrefix,
		translation.WithEnrichment(
			centralTranslation.New(mgr.GetClient()),
			proxy.NewProxyEnvVarsInjector(proxyEnv, mgr.GetLogger()),
			// Using uncached UncachedClient since this is reading secrets not
			// owned by the operator so we can't guarantee labels for cache
			// are set properly.
			legacy.NewImagePullSecretReferenceInjector(mgr.GetAPIReader(), "imagePullSecrets",
				"stackrox", "stackrox-scanner", "stackrox-scanner-v4")),
		opts...,
	)
}
