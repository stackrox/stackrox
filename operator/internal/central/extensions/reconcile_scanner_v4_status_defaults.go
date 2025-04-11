package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// This extension's sole purpose is to persist the Scanner V4 Enabled|Disabled setting
// in the Central status for later usage.
func ReconcileScannerV4StatusDefaultsExtension(client ctrlClient.Client, direct ctrlClient.Reader) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerV4StatusDefaults, client, direct)
}

func reconcileScannerV4StatusDefaults(ctx context.Context, central *platform.Central, _ ctrlClient.Client, _ ctrlClient.Reader, statusUpdater func(updateStatusFunc), logger logr.Logger) error {
	run := &scannerV4DefaultingExtensionRun{
		spec:          &central.Spec,
		statusUpdater: statusUpdater,
		logger:        logger,
	}
	return run.Execute(ctx)
}

type scannerV4DefaultingExtensionRun struct {
	spec          *platform.CentralSpec
	statusUpdater func(updateStatusFunc)
	logger        logr.Logger
}

func (r *scannerV4DefaultingExtensionRun) Execute(ctx context.Context) error {
	r.statusUpdater(r.updateStatus)
	return nil
}

func (r *scannerV4DefaultingExtensionRun) updateStatus(status *platform.CentralStatus) bool {
	// assert status != nil
	if status.Defaults == nil {
		status.Defaults = &platform.StatusDefaults{}
	}

	componentPolicy := defaulting.ScannerV4ComponentPolicy(status.Defaults, r.spec.ScannerV4)
	r.logger.Info("ScannerV4StatusDefaultsExtension computed componentPolicy", "componentPolicy", componentPolicy)
	status.Defaults.ScannerV4ComponentPolicy = string(componentPolicy)

	return true
}
