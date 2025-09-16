package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/internal/common/extensions"
	"github.com/stackrox/rox/operator/internal/securedcluster/scanner"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ commonExtensions.ScannerV4BearingCustomResource = (*securedClusterWithScannerV4Bearer)(nil)

type securedClusterWithScannerV4Bearer struct {
	*platform.SecuredCluster
	scannerV4Enabled bool
}

// IsScannerV4Enabled returns the scannerV4Enabled field
func (s *securedClusterWithScannerV4Bearer) IsScannerV4Enabled() bool {
	return s.scannerV4Enabled
}

// ReconcileLocalScannerV4DBPasswordExtension returns an extension that takes care of creating the scanner-v4-db-password
// secret ahead of time.
func ReconcileLocalScannerV4DBPasswordExtension(client ctrlClient.Client, direct ctrlClient.Reader) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerV4DBPassword, client, direct)
}

func reconcileScannerV4DBPassword(ctx context.Context, s *platform.SecuredCluster, client ctrlClient.Client, direct ctrlClient.Reader, _ func(updateStatusFunc), _ logr.Logger) error {
	config, err := scanner.AutoSenseLocalScannerV4Config(ctx, client, *s)
	if err != nil {
		return err
	}

	// Only reconcile password if resources are deployed with the SecuredCluster.
	securedClusterWithScannerV4 := &securedClusterWithScannerV4Bearer{
		SecuredCluster:   s,
		scannerV4Enabled: config.DeployScannerResources,
	}

	return commonExtensions.ReconcileScannerV4DBPassword(ctx, securedClusterWithScannerV4, client, direct)
}
