package extensions

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// CheckForbiddenNamespacesExtension returns an extension that verifies that the custom resource is not created
// in a forbidden namespace.
func CheckForbiddenNamespacesExtension(forbiddenNamespacePredicate func(namespace string) bool) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), _ logr.Logger) error {
		if obj.GetDeletionTimestamp() != nil {
			return nil
		}
		if forbiddenNamespacePredicate(obj.GetNamespace()) {
			return errors.Errorf("Namespace %q is forbidden as an operand namespace, as it should be reserved for system components. Please create this %s in a different, non-system namespace.", obj.GetNamespace(), obj.GroupVersionKind().Kind)
		}
		return nil
	}
}

// IsSystemNamespace checks if the given namespace is a system namespace.
func IsSystemNamespace(ns string) bool {
	return kubernetes.IsSystemNamespace(ns) || strings.HasPrefix(ns, "openshift-")
}
