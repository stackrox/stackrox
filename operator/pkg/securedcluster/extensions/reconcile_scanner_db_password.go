package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/securedcluster/scanner"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ commonExtensions.ScannerBearingCustomResource = (*securedClusterWithScannerBearer)(nil)

type securedClusterWithScannerBearer struct {
	*platform.SecuredCluster
	scannerEnabled bool
}

func (s *securedClusterWithScannerBearer) IsScannerEnabled() bool {
	return s.scannerEnabled
}

// ReconcileLocalScannerDBPasswordExtension returns an extension that takes care of creating the scanner-db-password
// secret ahead of time.
func ReconcileLocalScannerDBPasswordExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcile, client)
}

func reconcile(ctx context.Context, s *platform.SecuredCluster, client ctrlClient.Client, _ func(updateStatusFunc), _ logr.Logger) error {
	config, err := scanner.AutoSenseLocalScannerConfig(ctx, client, *s)
	if err != nil {
		return err
	}

	// Only reconcile password if resources are deployed with the SecuredCluster.
	securedClusterWithScanner := &securedClusterWithScannerBearer{
		SecuredCluster: s,
		scannerEnabled: config.DeployScannerResources,
	}
	return commonExtensions.ReconcileScannerDBPassword(ctx, securedClusterWithScanner, client)
}
