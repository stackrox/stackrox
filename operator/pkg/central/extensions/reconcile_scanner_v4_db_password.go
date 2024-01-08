package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_ commonExtensions.ScannerV4BearingCustomResource = (*platform.Central)(nil)
)

const (
	scannerV4DBPasswordKey = `password`
)

// ReconcileScannerV4DBPasswordExtension returns an extension that takes care of creating the scanner-v4-db-password
// secret ahead of time.
func ReconcileScannerV4DBPasswordExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerV4DBPassword, client)
}

func reconcileScannerV4DBPassword(ctx context.Context, c *platform.Central, client ctrlClient.Client, _ func(updateStatusFunc), _ logr.Logger) error {
	return commonExtensions.ReconcileScannerV4DBPassword(ctx, c, client)
}
