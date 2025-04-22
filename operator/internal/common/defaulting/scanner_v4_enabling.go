package defaulting

import (
	"fmt"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

var (
	defaultForUpgrades         = platform.ScannerV4Disabled
	defaultForNewInstallations = platform.ScannerV4Enabled
)

// Only returns Enabled or Disabled.
// Derive component policy based on status Defaults and spec.
// This will be called from the preExtension to record the current setting.
func ScannerV4ComponentPolicy(logger logr.Logger, status *platform.CentralStatus, spec *platform.ScannerV4Spec) platform.ScannerV4ComponentPolicy {
	if spec != nil && spec.ScannerComponent != nil {
		comp := *spec.ScannerComponent
		if comp == platform.ScannerV4ComponentEnabled || comp == platform.ScannerV4ComponentDisabled {
			logger.Info(fmt.Sprintf("ScannerV4ComponentPolicy: usingScannerV4 componentPolicy %v set in CR", comp))
			return comp
		}
	}

	// User is relying on defaulting.
	// This includes the case spec.ScannerComponent == "Default".

	if status == nil {
		// Install / Green field.
		logger.Info("ScannerV4ComponentPolicy: assuming new installation due to empty status.")
		return defaultForNewInstallations
	}

	if status.DeployedRelease == nil {
		// It seems that even though status is not nil, a previous installation attempt was not successful, hence we still
		// assume that the current reconcilliation run is a fresh installation.
		logger.Info("ScannerV4ComponentPolicy: assuming new installation due to empty deployedRelease status.")
		return defaultForNewInstallations
	}

	// Upgrade.
	logger.Info("ScannerV4ComponentPolicy: assuming upgrade.")

	if status.Defaults == nil || status.Defaults.ScannerV4ComponentPolicy == "" {
		// No entry in the statusDefaults yet -> preserve defaulting behavior of versions which did not populate
		// statusDefaults with a ScannerV4ComponentPolicy.
		logger.Info(fmt.Sprintf("ScannerV4ComponentPolicy: using ScannerV4 componentPolicy %v.", defaultForUpgrades))
		return defaultForUpgrades
	}

	// A default entry exists already, use recorded value.
	recordedDefault := platform.ScannerV4ComponentPolicy(status.Defaults.ScannerV4ComponentPolicy)
	if recordedDefault == platform.ScannerV4Enabled || recordedDefault == platform.ScannerV4Disabled {
		logger.Info(fmt.Sprintf("ScannerV4ComponentPolicy: using previously recorded ScannerV4 componentPolicy %v.", recordedDefault))
		return recordedDefault
	}

	// This should not happen, since we only store Enabled|Disabled, but just in case something unexpected happened
	// and we need to make some decisions...
	logger.Info(fmt.Sprintf("ScannerV4ComponentPolicy: detected previously recorded ScannerV4 componentPolicy %v, using %v instead.", recordedDefault, defaultForUpgrades))
	return defaultForUpgrades
}

// Convenience for some callers.
func ScannerV4ComponentPolicyEnabled(status *platform.CentralStatus, spec *platform.ScannerV4Spec) bool {
	componentPolicy := ScannerV4ComponentPolicy(logr.Discard(), status, spec)
	return componentPolicy == platform.ScannerV4ComponentEnabled
}
