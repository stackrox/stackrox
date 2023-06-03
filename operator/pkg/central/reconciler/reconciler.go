package reconciler

import (
	"context"

	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	pauseReconcileAnnotation = "stackrox.io/pause-reconcile"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager, selector string) error {

	//proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	opts := []pkgReconciler.Option{
		//pkgReconciler.WithExtraWatch(
		//	&source.Kind{Type: &platform.SecuredCluster{}},
		//	reconciler.HandleSiblings(platform.CentralGVK, mgr),
		//	Only appearance and disappearance of a SecuredCluster resource can influence whether
		//	an init bundle should be created by the Central controller.
		//utils.CreateAndDeleteOnlyPredicate{},
		//),

		//pkgReconciler.WithReconcilePeriod(extensions.InitBundleReconcilePeriod),
		//pkgReconciler.WithPreExtension(extensions.ReconcileCentralTLSExtensions(mgr.GetClient())),
		//pkgReconciler.WithPreExtension(extensions.ReconcileCentralDBPasswordExtension(mgr.GetClient())),
		//pkgReconciler.WithPreExtension(extensions.ReconcileScannerDBPasswordExtension(mgr.GetClient())),
		//pkgReconciler.WithPreExtension(extensions.ReconcileAdminPasswordExtension(mgr.GetClient())),
		//pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), extensions.PVCTargetCentral, extensions.DefaultCentralPVCName)),
		//pkgReconciler.WithPreExtension(extensions.ReconcilePVCExtension(mgr.GetClient(), extensions.PVCTargetCentralDB, extensions.DefaultCentralDBPVCName)),
		//pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), proxyEnv)),
		//pkgReconciler.WithPreExtension(commonExtensions.CheckForbiddenNamespacesExtension(commonExtensions.IsSystemNamespace)),
		//pkgReconciler.WithPreExtension(commonExtensions.ReconcileProductVersionStatusExtension(version.GetMainVersion())),
		//pkgReconciler.WithPauseReconcileAnnotation(pauseReconcileAnnotation),
	}

	opts, err := addSelectorOptionIfNeeded(selector, opts)
	if err != nil {
		return err
	}

	return reconciler.SetupReconcilerWithManager(
		mgr, platform.CentralGVK, image.CentralServicesChartPrefix,
		//proxy.InjectProxyEnvVars(translation.Translator{}, proxyEnv),
		&EmptyTranslator{},
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

type EmptyTranslator struct{}

func (e EmptyTranslator) Translate(ctx context.Context, unstructured *unstructured.Unstructured) (chartutil.Values, error) {
	// TODO: Pass Helm values from CR annotation, CR field (extra CR) or ConfigMap
	//if val, ok := unstructured.GetAnnotations()["helm-values"]; ok {
	//	return val
	//}
	return chartutil.Values{
		"central": chartutil.Values{
			"resources": chartutil.Values{
				"limits": chartutil.Values{
					"cpu": 100,
				},
			},
		},
	}, nil
}
