package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type updateStatusFunc func(status *platform.SecuredClusterStatus) bool

var (
	errUnexpectedGVK = errors.New("invoked reconciliation extension for object with unexpected GVK")
)

func wrapExtension(runFn func(ctx context.Context, securedCluster *platform.SecuredCluster, client ctrlClient.Client, direct ctrlClient.Reader, statusUpdater func(statusFunc updateStatusFunc), log logr.Logger) error, client ctrlClient.Client, direct ctrlClient.Reader) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, statusUpdater func(extensions.UpdateStatusFunc), log logr.Logger) error {
		if u.GroupVersionKind() != platform.SecuredClusterGVK {
			log.Error(errUnexpectedGVK, "unable to reconcile secured cluster", "expectedGVK", platform.SecuredClusterGVK, "actualGVK", u.GroupVersionKind())
			return errUnexpectedGVK
		}

		c := platform.SecuredCluster{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &c)
		if err != nil {
			return errors.Wrap(err, "converting object to SecuredCluster")
		}

		wrappedStatusUpdater := func(typedUpdateStatus updateStatusFunc) {
			statusUpdater(func(uSt *unstructured.Unstructured) bool {
				var status platform.SecuredClusterStatus
				_ = runtime.DefaultUnstructuredConverter.FromUnstructured(uSt.Object, &status)
				if !typedUpdateStatus(&status) {
					return false
				}
				uStNew, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&status)
				uSt.Object = uStNew
				return true
			})
		}
		return runFn(ctx, &c, client, direct, wrappedStatusUpdater, log)
	}
}
