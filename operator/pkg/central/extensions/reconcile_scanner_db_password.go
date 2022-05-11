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
	_ commonExtensions.ScannerBearingCustomResource = (*platform.Central)(nil)
)

const (
	scannerDBPasswordKey = `password`
)

// ReconcileScannerDBPasswordExtension returns an extension that takes care of creating the scanner-db-password
// secret ahead of time.
func ReconcileScannerDBPasswordExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerDBPassword, client)
}

func reconcileScannerDBPassword(ctx context.Context, c *platform.Central, client ctrlClient.Client, _ func(updateStatusFunc), _ logr.Logger) error {
	return commonExtensions.ReconcileScannerDBPassword(ctx, c, client)
}
