package extensions

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// This extension's purpose is to
//
//   1. apply defaults by mutating the Central spec as a prerequisite for the value translator
//   2. persist any implicit Scanner V4 Enabled|Disabled setting in the Central status for later usage during upgrade-reconcilliations.
//

func ReconcileScannerV4StatusDefaultsExtension() extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerV4StatusDefaults, nil, nil)
}

func reconcileScannerV4StatusDefaults(
	ctx context.Context, central *platform.Central, _ ctrlClient.Client, _ ctrlClient.Reader,
	registerStatusUpdater func(updateStatusFunc), logger logr.Logger) error {
	// assert central != nil
	if central.Spec.ScannerV4 == nil {
		central.Spec.ScannerV4 = &platform.ScannerV4Spec{}
	}
	scannerV4Spec := central.Spec.ScannerV4

	var scannerComp platform.ScannerV4ComponentPolicy
	if scannerV4Spec.ScannerComponent != nil {
		scannerComp = *scannerV4Spec.ScannerComponent
	}

	if scannerComp == platform.ScannerV4ComponentEnabled || scannerComp == platform.ScannerV4ComponentDisabled {
		// User provided an explicit choice, nothing to do in this extension.
		return nil
	}

	// User is relying on defaults. Compute default and register status-updating callback.
	status := &central.Status
	if reflect.DeepEqual(status, platform.CentralStatus{}) {
		status = nil
	}

	componentPolicy := defaulting.ScannerV4ComponentPolicy(logger, status, scannerV4Spec)
	scannerV4Spec.ScannerComponent = &componentPolicy // Mutate spec for the translator.

	registerStatusUpdater(func(status *platform.CentralStatus) bool {
		// Here we persist the default setting of the current operator version in the status object for usage
		// during a future reconcilliation.
		if status.Defaults == nil {
			status.Defaults = &platform.StatusDefaults{}
		}
		if status.Defaults.ScannerV4ComponentPolicy == string(componentPolicy) {
			return false
		}
		status.Defaults.ScannerV4ComponentPolicy = string(componentPolicy)

		return true
	})

	return nil
}
