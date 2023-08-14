package reconciler

import (
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/central/extensions"
	"github.com/stackrox/rox/operator/pkg/central/values/translation"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/proxy"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	"github.com/stackrox/rox/operator/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	pauseReconcileAnnotation = "stackrox.io/pause-reconcile"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager, selector string) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	opts := []pkgReconciler.Option{
		pkgReconciler.WithExtraWatch(
			source.Kind(mgr.GetCache(), &platform.SecuredCluster{}),
			reconciler.HandleSiblings(platform.CentralGVK, mgr),
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
		pkgReconciler.WithPauseReconcileAnnotation(pauseReconcileAnnotation),
	}

	opts, err := addSelectorOptionIfNeeded(selector, opts)
	if err != nil {
		return err
	}

	opts = commonExtensions.AddMapKubeAPIsExtensionIfMapFileExists(opts)

	return reconciler.SetupReconcilerWithManager(
		mgr, platform.CentralGVK, image.CentralServicesChartPrefix,
		proxy.InjectProxyEnvVars(translation.New(mgr.GetClient()), proxyEnv, mgr.GetLogger()),
		opts...,
	)
}

func addSelectorOptionIfNeeded(selector string, opts []pkgReconciler.Option) ([]pkgReconciler.Option, error) {
	if len(selector) == 0 {
		return opts, nil
	}
	labelSelector, err := v1.ParseToLabelSelector(selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse central label selector")
	}
	if labelSelector != nil {
		ctrl.Log.Info("Using central label selector", "selector", selector)
		opts = append(opts, pkgReconciler.WithSelector(*labelSelector))
	}
	return opts, nil
}
