package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	securedClusterv1Alpha1 "github.com/stackrox/rox/operator/api/securedcluster/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type updateStatusFunc func(status *securedClusterv1Alpha1.SecuredClusterStatus) bool

var (
	errUnexpectedGVK = errors.New("invoked reconciliation extension for object with unexpected GVK")
)

func wrapExtension(runFn func(ctx context.Context, securedCluster *securedClusterv1Alpha1.SecuredCluster, k8sClient kubernetes.Interface, statusUpdater func(statusFunc updateStatusFunc), log logr.Logger) error, k8sClient kubernetes.Interface) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, statusUpdater func(extensions.UpdateStatusFunc), log logr.Logger) error {
		if u.GroupVersionKind() != securedClusterv1Alpha1.SecuredClusterGVK {
			log.Error(errUnexpectedGVK, "unable to reconcile secured cluster", "expectedGVK", securedClusterv1Alpha1.SecuredClusterGVK, "actualGVK", u.GroupVersionKind())
			return errUnexpectedGVK
		}

		c := securedClusterv1Alpha1.SecuredCluster{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &c)
		if err != nil {
			return errors.Wrap(err, "converting object to SecuredCluster")
		}

		wrappedStatusUpdater := func(typedUpdateStatus updateStatusFunc) {
			statusUpdater(func(uSt *unstructured.Unstructured) bool {
				var status securedClusterv1Alpha1.SecuredClusterStatus
				_ = runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &status)
				if !typedUpdateStatus(&status) {
					return false
				}
				uStNew, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&status)
				uSt.Object = uStNew
				return true
			})
		}
		return runFn(ctx, &c, k8sClient, wrappedStatusUpdater, log)
	}
}
