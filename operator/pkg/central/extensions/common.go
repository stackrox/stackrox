package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type updateStatusFunc func(*platform.CentralStatus) bool

var (
	errUnexpectedGVK = errors.New("invoked reconciliation extension for object with unexpected GVK")
)

func wrapExtension(runFn func(ctx context.Context, central *platform.Central, client ctrlClient.Client, statusUpdater func(statusFunc updateStatusFunc), log logr.Logger) error, client ctrlClient.Client) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, statusUpdater func(extensions.UpdateStatusFunc), log logr.Logger) error {
		if u.GroupVersionKind() != platform.CentralGVK {
			log.Error(errUnexpectedGVK, "unable to reconcile central", "expectedGVK", platform.CentralGVK, "actualGVK", u.GroupVersionKind())
			return errUnexpectedGVK
		}

		c := platform.Central{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &c)
		if err != nil {
			return errors.Wrap(err, "converting object to Central")
		}

		wrappedStatusUpdater := func(typedUpdateStatus updateStatusFunc) {
			statusUpdater(func(uSt *unstructured.Unstructured) bool {
				var status platform.CentralStatus
				_ = runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &status)
				if !typedUpdateStatus(&status) {
					return false
				}
				uStNew, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&status)
				uSt.Object = uStNew
				return true
			})
		}
		return runFn(ctx, &c, client, wrappedStatusUpdater, log)
	}
}
