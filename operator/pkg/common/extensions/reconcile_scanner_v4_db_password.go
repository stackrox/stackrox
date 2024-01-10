package extensions

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/pkg/renderer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ScannerV4DBPasswordKey          = `password`
	scannerV4DBPasswordResourceName = "scanner-v4-db-password" // #nosec G101
)

// ScannerV4BearingCustomResource interface exposes details about the Scanner resource from the kubernetes object.
type ScannerV4BearingCustomResource interface {
	types.K8sObject
	IsScannerV4Enabled() bool
}

// ReconcileScannerV4DBPassword reconciles a scanner v4 db password
func ReconcileScannerV4DBPassword(ctx context.Context, obj ScannerV4BearingCustomResource, client ctrlClient.Client) error {
	return reconcileScannerV4DBPassword(ctx, obj, client)
}

func reconcileScannerV4DBPassword(ctx context.Context, obj ScannerV4BearingCustomResource, client ctrlClient.Client) error {
	run := &reconcileScannerV4DBPasswordExtensionRun{
		SecretReconciliator:  NewSecretReconciliator(client, obj),
		obj:                  obj,
		passwordResourceName: scannerV4DBPasswordResourceName,
	}
	return run.Execute(ctx)
}

type reconcileScannerV4DBPasswordExtensionRun struct {
	*SecretReconciliator
	obj                  ScannerV4BearingCustomResource
	passwordResourceName string
}

func (r *reconcileScannerV4DBPasswordExtensionRun) Execute(ctx context.Context) error {
	// Delete any scanner-db password only if the CR is being deleted, or scanner is not enabled.
	shouldExist := r.obj.GetDeletionTimestamp() == nil && r.obj.IsScannerV4Enabled()

	if err := r.reconcilePasswordSecret(ctx, shouldExist); err != nil {
		return errors.Wrapf(err, "reconciling %q secret", r.passwordResourceName)
	}

	return nil
}

func (r *reconcileScannerV4DBPasswordExtensionRun) reconcilePasswordSecret(ctx context.Context, shouldExist bool) error {
	if shouldExist {
		return r.EnsureSecret(ctx, r.passwordResourceName, r.validateScannerV4DBPasswordData, r.generateScannerV4DBPasswordData, true)
	}
	return r.DeleteSecret(ctx, r.passwordResourceName)
}

func (r *reconcileScannerV4DBPasswordExtensionRun) validateScannerV4DBPasswordData(data types.SecretDataMap, _ bool) error {
	if len(data[ScannerV4DBPasswordKey]) == 0 {
		return errors.Errorf("%s secret must contain a non-empty %q entry", r.passwordResourceName, ScannerV4DBPasswordKey)
	}
	return nil
}

func (r *reconcileScannerV4DBPasswordExtensionRun) generateScannerV4DBPasswordData() (types.SecretDataMap, error) {
	data := types.SecretDataMap{
		ScannerV4DBPasswordKey: []byte(renderer.CreatePassword()),
	}
	return data, nil
}
