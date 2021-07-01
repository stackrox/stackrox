package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	centralv1Alpha1 "github.com/stackrox/rox/operator/api/central/v1alpha1"
	"github.com/stackrox/rox/pkg/renderer"
	"k8s.io/client-go/kubernetes"
)

const (
	scannerDBPasswordKey = `password`
)

// ReconcileScannerDBPasswordExtension returns an extension that takes care of creating the scanner-db-password
// secret ahead of time.
func ReconcileScannerDBPasswordExtension(k8sClient kubernetes.Interface) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerDBPassword, k8sClient)
}

func reconcileScannerDBPassword(ctx context.Context, c *centralv1Alpha1.Central, k8sClient kubernetes.Interface, log logr.Logger) error {
	run := &reconcileScannerDBPasswordExtensionRun{
		secretReconciliationExtension: secretReconciliationExtension{
			ctx:        ctx,
			k8sClient:  k8sClient,
			centralObj: c,
		},
	}
	return run.Execute()
}

type reconcileScannerDBPasswordExtensionRun struct {
	secretReconciliationExtension
}

func (r *reconcileScannerDBPasswordExtensionRun) Execute() error {
	// Delete any scanner-db password only if the CR is being deleted, or scanner is not enabled.
	shouldDelete := r.centralObj.DeletionTimestamp != nil || !r.centralObj.Spec.Scanner.IsEnabled()

	if err := r.reconcileSecret("scanner-db-password", !shouldDelete, r.validateScannerDBPasswordData, r.generateScannerDBPasswordData); err != nil {
		return errors.Wrap(err, "reconciling scanner-db-password secret")
	}

	return nil
}

func (r *reconcileScannerDBPasswordExtensionRun) validateScannerDBPasswordData(data secretDataMap) error {
	if len(data[scannerDBPasswordKey]) == 0 {
		return errors.Errorf("scanner-db-password secret must contain a non-empty %q entry", scannerDBPasswordKey)
	}
	return nil
}

func (r *reconcileScannerDBPasswordExtensionRun) generateScannerDBPasswordData() (secretDataMap, error) {
	data := secretDataMap{
		scannerDBPasswordKey: []byte(renderer.CreatePassword()),
	}
	return data, nil
}
