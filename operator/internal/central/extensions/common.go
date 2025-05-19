package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/reflectutils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type updateStatusFunc func(*platform.CentralStatus) bool

var (
	errUnexpectedGVK = errors.New("invoked reconciliation extension for object with unexpected GVK")
)

func wrapExtension(runFn func(ctx context.Context, central *platform.Central, client ctrlClient.Client, direct ctrlClient.Reader, statusUpdater func(statusFunc updateStatusFunc), log logr.Logger) error, client ctrlClient.Client, direct ctrlClient.Reader) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, statusUpdater func(extensions.UpdateStatusFunc), log logr.Logger) error {
		if u.GroupVersionKind() != platform.CentralGVK {
			log.Error(errUnexpectedGVK, "unable to reconcile central", "expectedGVK", platform.CentralGVK, "actualGVK", u.GroupVersionKind())
			return errUnexpectedGVK
		}

		// Convert unstructured object into Central.
		c := platform.Central{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &c)
		if err != nil {
			return errors.Wrap(err, "converting object to Central")
		}

		// refactor!
		// For translation purposes, enrich Central with defaults, which are not implicitly marshalled/unmarshaled.
		// and merge into spec to make them visible to the extensions (e.g. for creating scanner-v4-db-password).
		centralDefaults := platform.CentralSpec{}
		if defaultsObj, ok := u.Object["defaults"].(map[string]interface{}); ok {
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(defaultsObj, &centralDefaults)
			if err != nil {
				return errors.Wrap(err, "converting defaults from unstructured object")
			}
		}
		c.Defaults = centralDefaults
		mergeDefaultsIntoSpec(&c)

		// Manually add the Central defaults as well.
		if defaults, ok := u.Object["defaults"].(map[string]interface{}); ok {
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(defaults, &c.Defaults)
			if err != nil {
				return errors.Wrap(err, "copying defaults from unstructured object into Central object")
			}
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
		err = runFn(ctx, &c, client, direct, wrappedStatusUpdater, log)
		if err != nil {
			return err
		}

		return nil
	}
}

// refactor!
func mergeDefaultsIntoSpec(central *platform.Central) {
	specWithDefaults := reflectutils.DeepMergeStructs(central.Defaults, central.Spec).(platform.CentralSpec)
	central.Spec = specWithDefaults
}
