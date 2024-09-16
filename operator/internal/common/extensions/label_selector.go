package extensions

import (
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// AddSelectorOptionIfNeeded conditionally adds label selector to opts for reconciler
func AddSelectorOptionIfNeeded(selector string, opts []pkgReconciler.Option) ([]pkgReconciler.Option, error) {
	if len(selector) == 0 {
		return opts, nil
	}
	labelSelector, err := v1.ParseToLabelSelector(selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse label selector")
	}
	if labelSelector != nil {
		ctrl.Log.Info("using label selector", "selector", selector)
		opts = append(opts, pkgReconciler.WithSelector(*labelSelector))
	}
	return opts, nil
}
