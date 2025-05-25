package reconciler

import (
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/central/common"
	"github.com/stackrox/rox/operator/internal/central/extensions"
	centralTranslation "github.com/stackrox/rox/operator/internal/central/values/translation"
	commonExtensions "github.com/stackrox/rox/operator/internal/common/extensions"
	"github.com/stackrox/rox/operator/internal/legacy"
	"github.com/stackrox/rox/operator/internal/proxy"
	"github.com/stackrox/rox/operator/internal/reconciler"
	"github.com/stackrox/rox/operator/internal/route"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/operator/internal/values/translation"
	"github.com/stackrox/rox/pkg/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager, selector string) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	extraEventWatcher := pkgReconciler.WithExtraWatch(
		source.Kind[*platform.SecuredCluster](
			mgr.GetCache(),
			&platform.SecuredCluster{},
			reconciler.HandleSiblings[*platform.SecuredCluster](platform.CentralGVK, mgr),
			// Only appearance and disappearance of a SecuredCluster resource can influence whether
			// an init bundle should be created by the Central controller.
			utils.CreateAndDeleteOnlyPredicate[*platform.SecuredCluster]{}))
	// IMPORTANT: The ReconcilerExtensionFeatureDefaulting preExtensions implements feature-defaulting logic
	// and therefore must be executed and registered first.
	// New extensions shall be added to otherPreExtensions to guarantee this ordering.
	otherPreExtensions := []pkgReconciler.Option{
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralTLSExtensions(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileCentralDBPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileScannerDBPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileScannerV4DBPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileAdminPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), mgr.GetAPIReader(), extensions.PVCTargetCentral, extensions.DefaultCentralPVCName)),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), mgr.GetAPIReader(), extensions.PVCTargetCentralDB, extensions.DefaultCentralDBPVCName)),
		pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), mgr.GetAPIReader(), extensions.PVCTargetCentralDBBackup, common.DefaultCentralDBBackupPVCName, extensions.WithDefaultClaimSize(extensions.DefaultBackupPVCSize))),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), mgr.GetAPIReader(), proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
	}

	opts := make([]pkgReconciler.Option, 0, len(otherPreExtensions)+6)
	opts = append(opts, extraEventWatcher)
	opts = append(opts, pkgReconciler.WithPreExtension(extensions.FeatureDefaultingExtension(mgr.GetClient())))
	opts = append(opts, otherPreExtensions...)
	opts = append(opts, pkgReconciler.WithReconcilePeriod(extensions.InitBundleReconcilePeriod))
	opts = append(opts, pkgReconciler.WithPauseReconcileAnnotation(commonExtensions.PauseReconcileAnnotation))
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
				"stackrox", "stackrox-scanner", "stackrox-scanner-v4"),
			route.NewRouteInjector(mgr.GetClient(), mgr.GetAPIReader(), mgr.GetLogger()),
		),
		opts...,
	)
}
