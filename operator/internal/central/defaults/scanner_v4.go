package defaults

import (
	"reflect"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common"
)

var (
	CentralScannerV4DefaultingFlow = CentralDefaultingFlow{
		Name:           "scanner-V4",
		DefaultingFunc: centralScannerV4Defaulting,
	}
)

// Only returns Enabled or Disabled.
// Derive component policy based on status Defaults and spec.
// This will be called from the preExtension to record the current setting.
//
// Second return value is `true`, if defaulting has been applied due to lack of explicit setting.
func CentralScannerV4ComponentPolicy(logger logr.Logger, status *platform.CentralStatus, annotations map[string]string, spec *platform.ScannerV4Spec) (platform.ScannerV4ComponentPolicy, bool) {
	defaultForUpgrades := platform.ScannerV4Disabled
	defaultForNewInstallations := platform.ScannerV4Enabled

	if spec != nil && spec.ScannerComponent != nil {
		comp := *spec.ScannerComponent
		if comp == platform.ScannerV4ComponentEnabled || comp == platform.ScannerV4ComponentDisabled {
			logger.Info("using ScannerV4 componentPolicy set in CR", "componentPolicy", comp)
			return comp, false
		}
	}

	// User is relying on defaulting.
	// This includes the case spec.ScannerComponent == "Default".

	// A default entry exists already, use recorded value.
	recordedValue := platform.ScannerV4ComponentPolicy(annotations[common.FeatureDefaultKeyScannerV4])
	if recordedValue == platform.ScannerV4Enabled || recordedValue == platform.ScannerV4Disabled {
		logger.Info("using previously recorded ScannerV4 componentPolicy", "componentPolicy", recordedValue)
		return recordedValue, true
	}

	// No or unexpected default set in the annotations.

	if isNewInstallation(status) {
		logger.Info("Assuming new installation due to incomplete status.")
		return defaultForNewInstallations, true
	}

	// Upgrade.
	logger.Info("assuming upgrade")

	if annotations[common.FeatureDefaultKeyScannerV4] == "" {
		// No entry in the statusDefaults yet -> preserve defaulting behavior of versions which did not populate
		// statusDefaults with a ScannerV4ComponentPolicy.
		logger.Info("empty feature-default annotation, using default ScannerV4 componentPolicy for upgrades",
			"componentPolicy", defaultForUpgrades)
		return defaultForUpgrades, true
	}

	// This should not happen, since we only store Enabled|Disabled, but just in case something unexpected happened
	// and we need to make some decisions...
	logger.Info("detected unexpected ScannerV4 componentPolicy in feature-default annotation, using default componentPolicy for upgrades",
		"unexpectedComponentPolicy", recordedValue,
		"componentPolicy", defaultForUpgrades)
	return defaultForUpgrades, true
}

// isNewInstallation checks if this is a new installation based on the status.
func isNewInstallation(status *platform.CentralStatus) bool {
	// The ProductVersion is only set post installation.
	return status == nil ||
		reflect.DeepEqual(status, &platform.CentralStatus{}) ||
		status.ProductVersion == ""
}

func centralScannerV4Defaulting(logger logr.Logger, status *platform.CentralStatus, annotations map[string]string, spec *platform.CentralSpec, defaults *platform.CentralSpec) error {
	scannerV4Spec := copyScannerV4Spec(spec.ScannerV4)
	componentPolicy, usedDefaulting := CentralScannerV4ComponentPolicy(logger, status, annotations, scannerV4Spec)
	if !usedDefaulting {
		// User provided an explicit choice, nothing to do in this flow.
		return nil
	}

	// User is relying on defaults. Set in-memory default and persist corresponding annotation.

	if annotations[common.FeatureDefaultKeyScannerV4] != string(componentPolicy) {
		// Update feature default setting.
		annotations[common.FeatureDefaultKeyScannerV4] = string(componentPolicy)
	}

	if defaults.ScannerV4 == nil {
		defaults.ScannerV4 = &platform.ScannerV4Spec{}
	}
	defaults.ScannerV4.ScannerComponent = &componentPolicy
	return nil
}

func copyScannerV4Spec(spec *platform.ScannerV4Spec) *platform.ScannerV4Spec {
	if spec == nil {
		return &platform.ScannerV4Spec{}
	}
	return spec.DeepCopy()
}
