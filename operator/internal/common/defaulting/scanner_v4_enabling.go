package defaulting

import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

var (
	defaultForUpgrades         = platform.ScannerV4Disabled
	defaultForNewInstallations = platform.ScannerV4Enabled
)

const (
	FeatureDefaultKeyScannerV4 = "feature-defaults.platform.stackrox.io/scannerV4"
)

// Only returns Enabled or Disabled.
// Derive component policy based on status Defaults and spec.
// This will be called from the preExtension to record the current setting.
//
// Second return value is `true`, if defaulting has been applied due to lack of explicit setting.
func ScannerV4ComponentPolicy(logger logr.Logger, status *platform.CentralStatus,
	annotations map[string]string, spec *platform.ScannerV4Spec) (platform.ScannerV4ComponentPolicy, bool) {
	if spec != nil && spec.ScannerComponent != nil {
		comp := *spec.ScannerComponent
		if comp == platform.ScannerV4ComponentEnabled || comp == platform.ScannerV4ComponentDisabled {
			logger.Info(fmt.Sprintf("ScannerV4ComponentPolicy: using ScannerV4 componentPolicy %v set in CR", comp))
			return comp, false
		}
	}

	// User is relying on defaulting.
	// This includes the case spec.ScannerComponent == "Default".

	// A default entry exists already, use recorded value.
	recordedDefault := platform.ScannerV4ComponentPolicy(annotations[FeatureDefaultKeyScannerV4])
	if recordedDefault == platform.ScannerV4Enabled || recordedDefault == platform.ScannerV4Disabled {
		logger.Info(fmt.Sprintf("ScannerV4ComponentPolicy: using previously recorded ScannerV4 componentPolicy %v.", recordedDefault))
		return recordedDefault, true
	}

	// No default set in the annotations.

	if centralStatusUninitialized(status) {
		// Install / Green field.
		logger.Info("ScannerV4ComponentPolicy: assuming new installation due to empty status.")
		return defaultForNewInstallations, true
	}

	// Upgrade.
	logger.Info("ScannerV4ComponentPolicy: assuming upgrade.")

	if annotations[FeatureDefaultKeyScannerV4] == "" {
		// No entry in the statusDefaults yet -> preserve defaulting behavior of versions which did not populate
		// statusDefaults with a ScannerV4ComponentPolicy.
		logger.Info(fmt.Sprintf("ScannerV4ComponentPolicy: using ScannerV4 componentPolicy %v.", defaultForUpgrades))
		return defaultForUpgrades, true
	}

	// This should not happen, since we only store Enabled|Disabled, but just in case something unexpected happened
	// and we need to make some decisions...
	logger.Info(fmt.Sprintf("ScannerV4ComponentPolicy: detected previously recorded ScannerV4 componentPolicy %v, using %v instead.", recordedDefault, defaultForUpgrades))
	return defaultForUpgrades, true
}

// centralStatusUninitialized checks if the provided Central status is uninitialized.
func centralStatusUninitialized(status *platform.CentralStatus) bool {
	return status == nil || reflect.DeepEqual(status, &platform.CentralStatus{})
}
