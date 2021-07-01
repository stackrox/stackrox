package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	centralv1Alpha1 "github.com/stackrox/rox/operator/api/central/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

// secretDataMap represents data stored as part of a secret.
type secretDataMap = map[string][]byte

type updateStatusFunc func(*centralv1Alpha1.CentralStatus) bool

var (
	errUnexpectedGVK = errors.New("invoked reconciliation extension for object with unexpected GVK")
)

func wrapExtension(runFn func(ctx context.Context, central *centralv1Alpha1.Central, k8sClient kubernetes.Interface, statusUpdater func(statusFunc updateStatusFunc), log logr.Logger) error, k8sClient kubernetes.Interface) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, statusUpdater func(extensions.UpdateStatusFunc), log logr.Logger) error {
		if u.GroupVersionKind() != centralv1Alpha1.CentralGVK {
			log.Error(errUnexpectedGVK, "unable to reconcile central TLS secrets", "expectedGVK", centralv1Alpha1.CentralGVK, "actualGVK", u.GroupVersionKind())
			return errUnexpectedGVK
		}

		c := centralv1Alpha1.Central{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &c)
		if err != nil {
			return errors.Wrap(err, "converting object to Central")
		}

		wrappedStatusUpdater := func(typedUpdateStatus updateStatusFunc) {
			statusUpdater(func(uSt *unstructured.Unstructured) bool {
				var status centralv1Alpha1.CentralStatus
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
