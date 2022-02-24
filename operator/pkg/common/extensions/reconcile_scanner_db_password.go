package extensions

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/pkg/renderer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	scannerDBPasswordKey = `password`
	passwordResourceName = "scanner-db-password"
)

type k8sObjectWithScanner interface {
	types.K8sObject
	ScannerEnabled
}

// ScannerEnabled is implemented by k8s API structs providing information about whether the Scanner component is enabled.
// The interface must be implemented to be compatible with this reconciler.
type ScannerEnabled interface {
	ScannerEnabled() bool
}

// reconcileScannerDBPasswordConfig represents the config for scanner db password reconciliation
type reconcileScannerDBPasswordConfig struct {
	PasswordResourceName string
}

// ReconcileScannerDBPassword reconciles a scanner db password
func ReconcileScannerDBPassword(ctx context.Context, obj k8sObjectWithScanner, client ctrlClient.Client) error {
	return reconcileScannerDBPassword(ctx, obj, client, reconcileScannerDBPasswordConfig{
		PasswordResourceName: passwordResourceName,
	})
}

func reconcileScannerDBPassword(ctx context.Context, obj k8sObjectWithScanner, client ctrlClient.Client, config reconcileScannerDBPasswordConfig) error {
	run := &reconcileScannerDBPasswordExtensionRun{
		SecretReconciliator:  NewSecretReconciliator(ctx, client, obj),
		obj:                  obj,
		passwordResourceName: config.PasswordResourceName,
	}
	return run.Execute()
}

type reconcileScannerDBPasswordExtensionRun struct {
	*SecretReconciliator
	obj                  k8sObjectWithScanner
	passwordResourceName string
	scannerIsEnabled     bool
}

func (r *reconcileScannerDBPasswordExtensionRun) Execute() error {
	// Delete any scanner-db password only if the CR is being deleted, or scanner is not enabled.
	shouldDelete := r.obj.GetDeletionTimestamp() != nil || !r.obj.ScannerEnabled()

	if err := r.ReconcileSecret(r.passwordResourceName, !shouldDelete, r.validateScannerDBPasswordData, r.generateScannerDBPasswordData, true); err != nil {
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
