package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ValidateDeploymentDefaultsExtension validates that deploymentDefaults configuration is valid
// for both Central and SecuredCluster CRs
func ValidateDeploymentDefaultsExtension() extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, _ func(extensions.UpdateStatusFunc), logger logr.Logger) error {
		logger = logger.WithName("extension-validate-deployment-defaults")

		if u.GetDeletionTimestamp() != nil {
			logger.Info("skipping validation due to deletionTimestamp being present")
			return nil
		}

		customizeSpec, found, err := unstructured.NestedMap(u.Object, "spec", "customize")
		if err != nil {
			return err
		}
		if !found || customizeSpec == nil {
			return nil
		}

		var customize platform.CustomizeSpec
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(customizeSpec, &customize); err != nil {
			return err
		}

		if err := customize.ValidateDeploymentDefaults(); err != nil {
			logger.Error(err, "invalid deploymentDefaults configuration")
			return err
		}

		return nil
	}
}
