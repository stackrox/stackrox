package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler extension, which fails on initial reconcilliation, if a SecuredCluster CR with the same namespace/name exists.
func VerifyCollisionFreeCentral(client ctrlClient.Client) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, _ func(extensions.UpdateStatusFunc), l logr.Logger) error {
		return verifyCollisionFreeCentral(ctx, client, u, l)
	}
}

func verifyCollisionFreeCentral(ctx context.Context, client ctrlClient.Client, u *unstructured.Unstructured, logger logr.Logger) error {
	logger = logger.WithName("extension-collision-check")
	if u.GroupVersionKind() != platform.CentralGVK {
		logger.Error(errUnexpectedGVK, "GVK mismatch", "expectedGVK", platform.CentralGVK, "actualGVK", u.GroupVersionKind())
		return errUnexpectedGVK
	}

	if common.CustomResourceAlreadyReconciled(u) {
		return nil
	}

	err := common.VerifyResourceNonExistence(ctx, client, platform.SecuredClusterGVK, "securedclusters", u.GetNamespace(), u.GetName())
	if err != nil {
		logger.Info("Central resource name collides with existing SecuredCluster resource",
			"name", u.GetName(), "namespace", u.GetNamespace())
		return errors.Wrap(err, "Central resource name collides with existing SecuredCluster resource")
	}

	return nil
}
