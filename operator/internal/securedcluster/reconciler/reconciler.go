package reconciler

import (
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/internal/common/extensions"
	"github.com/stackrox/rox/operator/internal/legacy"
	"github.com/stackrox/rox/operator/internal/proxy"
	"github.com/stackrox/rox/operator/internal/reconciler"
	"github.com/stackrox/rox/operator/internal/securedcluster/extensions"
	scTranslation "github.com/stackrox/rox/operator/internal/securedcluster/values/translation"
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
		source.Kind[*platform.Central](
			mgr.GetCache(),
			&platform.Central{},
			reconciler.HandleSiblings[*platform.Central](platform.SecuredClusterGVK, mgr),
			// Only appearance and disappearance of a Central resource can influence whether
			// a local scanner should be deployed by the SecuredCluster controller.
			utils.CreateAndDeleteOnlyPredicate[*platform.Central]{}))

	otherPreExtensions := []pkgReconciler.Option{
		pkgReconciler.WithPreExtension(extensions.CheckClusterNameExtension()),
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), mgr.GetAPIReader(), proxyEnv)),
		pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
		pkgReconciler.WithPreExtension(extensions.ReconcileLocalScannerDBPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
		pkgReconciler.WithPreExtension(extensions.ReconcileLocalScannerV4DBPasswordExtension(mgr.GetClient(), mgr.GetAPIReader())),
	}

	// IMPORTANT: reconciler preExtensions that implement feature-defaulting logic (such as ReconcileScannerV4FeatureDefaultsExtension)
	// must be executed first. Hence, they need to be first in order when registering extensions using WithPreExtension!
	opts := make([]pkgReconciler.Option, 0, len(otherPreExtensions)+5)
	opts = append(opts, extraEventWatcher)
	opts = append(opts, pkgReconciler.WithPreExtension(extensions.ReconcileScannerV4FeatureDefaultsExtension(mgr.GetClient())))
	opts = append(opts, otherPreExtensions...)
	opts = append(opts, pkgReconciler.WithPauseReconcileAnnotation(commonExtensions.PauseReconcileAnnotation))
	opts, err := commonExtensions.AddSelectorOptionIfNeeded(selector, opts)
	if err != nil {
		return err
	}
	opts = commonExtensions.AddMapKubeAPIsExtensionIfMapFileExists(opts)

	// Using uncached UncachedClient since this is reading secrets not
	// owned by the operator so we can't guarantee labels for cache
	// are set properly.
	pullSecretRefInjector := legacy.NewImagePullSecretReferenceInjector(
		mgr.GetAPIReader(), "imagePullSecrets",
		"secured-cluster-services-main", "stackrox", "stackrox-scanner", "stackrox-scanner-v4")
	pullSecretRefInjector = pullSecretRefInjector.WithExtraImagePullSecrets(
		"mainImagePullSecrets", "secured-cluster-services-main", "stackrox")
	pullSecretRefInjector = pullSecretRefInjector.WithExtraImagePullSecrets(
		"collectorImagePullSecrets", "secured-cluster-services-collector", "stackrox", "collector-stackrox")

	return reconciler.SetupReconcilerWithManager(
		mgr, platform.SecuredClusterGVK,
		image.SecuredClusterServicesChartPrefix,
		translation.WithEnrichment(
			scTranslation.New(mgr.GetClient(), mgr.GetAPIReader()),
			proxy.NewProxyEnvVarsInjector(proxyEnv, mgr.GetLogger()),
			pullSecretRefInjector,
		),
		opts...,
	)
}
