package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/pkg/renderer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	scannerDBPasswordKey = `password`
)

// ReconcileScannerDBPasswordExtension returns an extension that takes care of creating the scanner-db-password
// secret ahead of time.
func ReconcileScannerDBPasswordExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerDBPassword, client)
}

func reconcileScannerDBPassword(ctx context.Context, c *platform.Central, client ctrlClient.Client, _ func(updateStatusFunc), log logr.Logger) error {
	run := &reconcileScannerDBPasswordExtensionRun{
		SecretReconciliator: commonExtensions.NewSecretReconciliator(client, c),
		centralObj:          c,
	}
	return run.Execute(ctx)
}

type reconcileScannerDBPasswordExtensionRun struct {
	*commonExtensions.SecretReconciliator
	centralObj *platform.Central
}

func (r *reconcileScannerDBPasswordExtensionRun) Execute(ctx context.Context) error {
	// Delete any scanner-db password only if the CR is being deleted, or scanner is not enabled.
	shouldDelete := r.centralObj.DeletionTimestamp != nil || !r.centralObj.Spec.Scanner.IsEnabled()

	if err := r.ReconcileSecret(ctx, "scanner-db-password", !shouldDelete, r.validateScannerDBPasswordData, r.generateScannerDBPasswordData, true); err != nil {
		return errors.Wrap(err, "reconciling scanner-db-password secret")
	}

	return nil
}

func (r *reconcileScannerDBPasswordExtensionRun) validateScannerDBPasswordData(data types.SecretDataMap, _ bool) error {
	if len(data[scannerDBPasswordKey]) == 0 {
		return errors.Errorf("scanner-db-password secret must contain a non-empty %q entry", scannerDBPasswordKey)
	}
	return nil
}

func (r *reconcileScannerDBPasswordExtensionRun) generateScannerDBPasswordData() (types.SecretDataMap, error) {
	data := types.SecretDataMap{
		scannerDBPasswordKey: []byte(renderer.CreatePassword()),
	}
	return data, nil
}
