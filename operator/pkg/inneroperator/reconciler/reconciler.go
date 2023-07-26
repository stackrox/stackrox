package reconciler

import (
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/central/values/translation"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/proxy"
	"github.com/stackrox/rox/operator/pkg/reconciler"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	pauseReconcileAnnotation = "stackrox.io/pause-reconcile"
)

// RegisterNewReconciler registers a new helm reconciler in the given k8s controller manager
func RegisterNewReconciler(mgr ctrl.Manager, selector string) error {
	proxyEnv := proxy.GetProxyEnvVars() // fix at startup time
	opts := []pkgReconciler.Option{
		pkgReconciler.WithPreExtension(proxy.ReconcileProxySecretExtension(mgr.GetClient(), proxyEnv)),
		pkgReconciler.WithPauseReconcileAnnotation(pauseReconcileAnnotation),
	}

	opts, err := addSelectorOptionIfNeeded(selector, opts)
	if err != nil {
		return err
	}

	opts = commonExtensions.AddMapKubeAPIsExtensionIfMapFileExists(opts)

	return reconciler.SetupReconcilerWithManager(
		mgr, platform.SecuredClusterGVK, image.OperatorChartPrefix,
		proxy.InjectProxyEnvVars(translation.New(mgr.GetClient()), proxyEnv),
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
