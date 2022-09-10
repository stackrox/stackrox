package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/renderer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralDBPasswordKey          = `password`
	centralDBPasswordResourceName = "central-db-password"
)

// ReconcileCentralDBPasswordExtension returns an extension that takes care of creating the central-db-password
// secret ahead of time.
func ReconcileCentralDBPasswordExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileCentralDBPassword, client)
}

func reconcileCentralDBPassword(ctx context.Context, c *platform.Central, client ctrlClient.Client, _ func(updateStatusFunc), _ logr.Logger) error {
	if !features.PostgresDatastore.Enabled() || c.Spec.Central.DB.Preexisting() {
		return nil
	}
	run := &reconcileCentralDBPasswordExtensionRun{
		SecretReconciliator: commonExtensions.NewSecretReconciliator(client, c),
		obj:                 c,
	}
	return run.Execute(ctx)
}

type reconcileCentralDBPasswordExtensionRun struct {
	*commonExtensions.SecretReconciliator
	obj *platform.Central
}

func (r *reconcileCentralDBPasswordExtensionRun) Execute(ctx context.Context) error {
	// Delete any central-db password only if the CR is being deleted
	shouldExist := r.obj.GetDeletionTimestamp() == nil

	if err := r.ReconcileSecret(ctx, centralDBPasswordResourceName, shouldExist, r.validateCentralDBPasswordData, r.generateCentralDBPasswordData, true); err != nil {
		return errors.Wrapf(err, "reconciling %q secret", centralDBPasswordResourceName)
	}

	return nil
}

func (r *reconcileCentralDBPasswordExtensionRun) validateCentralDBPasswordData(data types.SecretDataMap, _ bool) error {
	if len(data[centralDBPasswordKey]) == 0 {
		return errors.Errorf("%s secret must contain a non-empty %q entry", centralDBPasswordResourceName, centralDBPasswordKey)
	}
	return nil
}

func (r *reconcileCentralDBPasswordExtensionRun) generateCentralDBPasswordData() (types.SecretDataMap, error) {
	data := types.SecretDataMap{
		centralDBPasswordKey: []byte(renderer.CreatePassword()),
	}
	return data, nil
}
