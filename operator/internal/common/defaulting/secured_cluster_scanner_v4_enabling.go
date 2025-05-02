package defaulting

import (
	"reflect"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

var (
	securedClusterScannerV4defaultForUpgrades         = platform.LocalScannerV4ComponentDisabled
	securedClusterScannerV4defaultForNewInstallations = platform.LocalScannerV4ComponentAutoSense
)

const (
	FeatureDefaultKeySecuredClusterScannerV4 = "feature-defaults.platform.stackrox.io/scannerV4"
)

// Only returns Enabled or Disabled.
// Derive component policy based on status Defaults and spec.
// This will be called from the preExtension to record the current setting.
//
// Second return value is `true`, if defaulting has been applied due to lack of explicit setting.
func SecuredClusterScannerV4ComponentPolicy(logger logr.Logger, status *platform.SecuredClusterStatus,
	annotations map[string]string, spec *platform.LocalScannerV4ComponentSpec) (platform.LocalScannerV4ComponentPolicy, bool) {
	logger = logger.WithName("scanner-v4-defaulting")

	if spec != nil && spec.ScannerComponent != nil {
		comp := *spec.ScannerComponent
		if comp == platform.LocalScannerV4ComponentAutoSense || comp == platform.LocalScannerV4ComponentDisabled {
			logger.Info("using ScannerV4 componentPolicy set in CR", "componentPolicy", comp)
			return comp, false
		}
	}

	// User is relying on defaulting.
	// This includes the case spec.ScannerComponent == "Default".

	// A default entry exists already, use recorded value.
	recordedValue := platform.LocalScannerV4ComponentPolicy(annotations[FeatureDefaultKeySecuredClusterScannerV4])
	if recordedValue == platform.LocalScannerV4ComponentAutoSense || recordedValue == platform.LocalScannerV4ComponentDisabled {
		logger.Info("using previously recorded ScannerV4 componentPolicy", "componentPolicy", recordedValue)
		return recordedValue, true
	}

	// No or unexpected default set in the annotations.

	if securedClusterStatusUninitialized(status) {
		// Install / Green field.
		logger.Info("ScannerV4ComponentPolicy: assuming new installation due to empty status.")
		return securedClusterScannerV4defaultForNewInstallations, true
	}

	// Upgrade.

	if annotations[FeatureDefaultKeySecuredClusterScannerV4] == "" {
		// No entry in the statusDefaults yet -> preserve defaulting behavior of versions which did not populate
		// statusDefaults with a ScannerV4ComponentPolicy.
		logger.Info("empty feature-default annotation, using default ScannerV4 componentPolicy for upgrades",
			"componentPolicy", defaultForUpgrades)
		return securedClusterScannerV4defaultForUpgrades, true
	}

	// This should not happen, since we only store Enabled|Disabled, but just in case something unexpected happened
	// and we need to make some decisions...
	logger.Info("detected unexpected ScannerV4 componentPolicy in feature-default annotation, using default componentPolicy for upgrades",
		"unexpectedComponentPolicy", recordedValue,
		"componentPolicy", defaultForUpgrades)
	return securedClusterScannerV4defaultForUpgrades, true
}

// securedClusterStatusUninitialized checks if the provided SecuredClusterStatus is uninitialized.
func securedClusterStatusUninitialized(status *platform.SecuredClusterStatus) bool {
	return status == nil || reflect.DeepEqual(status, &platform.SecuredClusterStatus{})
}
