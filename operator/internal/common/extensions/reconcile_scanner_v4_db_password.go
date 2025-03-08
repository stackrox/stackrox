package extensions

import (
	"context"

	"github.com/pkg/errors"
	commonLabels "github.com/stackrox/rox/operator/internal/common/labels"
	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/pkg/renderer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ScannerV4DBPasswordKey is the key used in the secret data for Scanner V4 db password
	ScannerV4DBPasswordKey          = `password`
	scannerV4DBPasswordResourceName = "scanner-v4-db-password" // #nosec G101
)

// ScannerV4BearingCustomResource interface exposes details about the Scanner V4 resource from the kubernetes object.
type ScannerV4BearingCustomResource interface {
	types.K8sObject
	IsScannerV4Enabled() bool
}

// ReconcileScannerV4DBPassword reconciles a Scanner V4 db password
func ReconcileScannerV4DBPassword(ctx context.Context, obj ScannerV4BearingCustomResource, client ctrlClient.Client, direct ctrlClient.Reader) error {
	return reconcileScannerV4DBPassword(ctx, obj, client, direct)
}

func reconcileScannerV4DBPassword(ctx context.Context, obj ScannerV4BearingCustomResource, client ctrlClient.Client, direct ctrlClient.Reader) error {
	run := &reconcileScannerV4DBPasswordExtensionRun{
		SecretReconciliator:  NewSecretReconciliator(client, direct, obj),
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
	// Delete any scanner-v4-db password only if the CR is being deleted, or Scanner V4 is not enabled.
	shouldExist := r.obj.GetDeletionTimestamp() == nil && r.obj.IsScannerV4Enabled()

	if err := r.reconcilePasswordSecret(ctx, shouldExist); err != nil {
		return errors.Wrapf(err, "reconciling %q secret", r.passwordResourceName)
	}

	return nil
}

func (r *reconcileScannerV4DBPasswordExtensionRun) reconcilePasswordSecret(ctx context.Context, shouldExist bool) error {
	if shouldExist {
		return r.EnsureSecret(ctx, r.passwordResourceName, r.validateScannerV4DBPasswordData, r.generateScannerV4DBPasswordData, commonLabels.DefaultLabels())
	}
	return r.DeleteSecret(ctx, r.passwordResourceName)
}

func (r *reconcileScannerV4DBPasswordExtensionRun) validateScannerV4DBPasswordData(data types.SecretDataMap, _ bool) error {
	if len(data[ScannerV4DBPasswordKey]) == 0 {
		return errors.Errorf("%s secret must contain a non-empty %q entry", r.passwordResourceName, ScannerV4DBPasswordKey)
	}
	return nil
}

func (r *reconcileScannerV4DBPasswordExtensionRun) generateScannerV4DBPasswordData(_ types.SecretDataMap) (types.SecretDataMap, error) {
	data := types.SecretDataMap{
		ScannerV4DBPasswordKey: []byte(renderer.CreatePassword()),
	}
	return data, nil
}
