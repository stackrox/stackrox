package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/internal/common/extensions"
	"github.com/stackrox/rox/operator/internal/common/rendercache"
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
func ReconcileScannerDBPasswordExtension(client ctrlClient.Client, direct ctrlClient.Reader) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerDBPassword, client, direct, nil)
}

func reconcileScannerDBPassword(ctx context.Context, c *platform.Central, client ctrlClient.Client, direct ctrlClient.Reader, _ func(updateStatusFunc), _ logr.Logger, _ *rendercache.RenderCache) error {
	return commonExtensions.ReconcileScannerDBPassword(ctx, c, client, direct)
}
