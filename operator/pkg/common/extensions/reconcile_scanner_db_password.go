package extensions

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/pkg/renderer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	scannerDBPasswordKey          = `password`
	scannerDBPasswordResourceName = "scanner-db-password"
)

// ScannerBearingCustomResource interface exposes details about the Scanner resource from the kubernetes object.
type ScannerBearingCustomResource interface {
	types.K8sObject
	IsScannerEnabled() bool
}

// reconcileScannerDBPasswordConfig represents the config for scanner db password reconciliation
type reconcileScannerDBPasswordConfig struct {
	PasswordResourceName string
}

// ReconcileScannerDBPassword reconciles a scanner db password
func ReconcileScannerDBPassword(ctx context.Context, obj ScannerBearingCustomResource, client ctrlClient.Client) error {
	return reconcileScannerDBPassword(ctx, obj, client, reconcileScannerDBPasswordConfig{
		PasswordResourceName: scannerDBPasswordResourceName,
	})
}

func reconcileScannerDBPassword(ctx context.Context, obj ScannerBearingCustomResource, client ctrlClient.Client, config reconcileScannerDBPasswordConfig) error {
	run := &reconcileScannerDBPasswordExtensionRun{
		SecretReconciliator:  NewSecretReconciliator(client, obj),
		obj:                  obj,
		passwordResourceName: config.PasswordResourceName,
	}
	return run.Execute(ctx)
}

type reconcileScannerDBPasswordExtensionRun struct {
	*SecretReconciliator
	obj                  ScannerBearingCustomResource
	passwordResourceName string
}

func (r *reconcileScannerDBPasswordExtensionRun) Execute(ctx context.Context) error {
	// Delete any scanner-db password only if the CR is being deleted, or scanner is not enabled.
	shouldExist := r.obj.GetDeletionTimestamp() == nil && r.obj.IsScannerEnabled()

	if err := r.ReconcileSecret(ctx, r.passwordResourceName, shouldExist, r.validateScannerDBPasswordData, r.generateScannerDBPasswordData, true); err != nil {
		return errors.Wrapf(err, "reconciling %q secret", r.passwordResourceName)
	}

	return nil
}

func (r *reconcileScannerDBPasswordExtensionRun) validateScannerDBPasswordData(data types.SecretDataMap, _ bool) error {
	if len(data[scannerDBPasswordKey]) == 0 {
		return errors.Errorf("%s secret must contain a non-empty %q entry", r.passwordResourceName, scannerDBPasswordKey)
	}
	return nil
}

func (r *reconcileScannerDBPasswordExtensionRun) generateScannerDBPasswordData() (types.SecretDataMap, error) {
	data := types.SecretDataMap{
		scannerDBPasswordKey: []byte(renderer.CreatePassword()),
	}
	return data, nil
}
