package defaulting

import (
	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

var (
	securedClusterScannerDefaultForUpgrades         = platform.LocalScannerComponentDisabled
	securedClusterScannerDefaultForNewInstallations = platform.LocalScannerComponentAutoSense
)

const (
	FeatureDefaultKeySecuredClusterScanner = "feature-defaults.platform.stackrox.io/scanner"
)

// Only returns Enabled or Disabled.
// Derive component policy based on status Defaults and spec.
// This will be called from the preExtension to record the current setting.
//
// Second return value is `true`, if defaulting has been applied due to lack of explicit setting.
func SecuredClusterScannerComponentPolicy(logger logr.Logger, status *platform.SecuredClusterStatus,
	annotations map[string]string, spec *platform.LocalScannerComponentSpec) (platform.LocalScannerComponentPolicy, bool) {
	logger = logger.WithName("scanner-defaulting")

	if spec != nil && spec.ScannerComponent != nil {
		comp := *spec.ScannerComponent
		if comp == platform.LocalScannerComponentAutoSense || comp == platform.LocalScannerComponentDisabled {
			logger.Info("using Scanner componentPolicy set in custom resource", "componentPolicy", comp)
			return comp, false
		}
	}

	// User is relying on defaulting.
	// This includes the case spec.ScannerComponent == "Default".

	// A default entry exists already, use recorded value.
	recordedValue := platform.LocalScannerComponentPolicy(annotations[FeatureDefaultKeySecuredClusterScanner])
	if recordedValue == platform.LocalScannerComponentAutoSense || recordedValue == platform.LocalScannerComponentDisabled {
		logger.Info("using previously recorded Scanner componentPolicy", "componentPolicy", recordedValue)
		return recordedValue, true
	}

	// No or unexpected default set in the annotations.

	if securedClusterStatusUninitialized(status) {
		// Install / Green field.
		logger.Info("assuming new installation due to empty status.")
		return securedClusterScannerDefaultForNewInstallations, true
	}

	// Upgrade.

	if annotations[FeatureDefaultKeySecuredClusterScanner] == "" {
		// No entry in the statusDefaults yet -> preserve defaulting behavior of versions which did not populate
		// statusDefaults with a LocalScannerComponentPolicy.
		logger.Info("empty feature-default annotation, using default Scanner componentPolicy for upgrades",
			"componentPolicy", defaultForUpgrades)
		return securedClusterScannerDefaultForUpgrades, true
	}

	// This should not happen, since we only store Enabled|Disabled, but just in case something unexpected happened
	// and we need to make some decisions...
	logger.Info("detected unexpected Scanner componentPolicy in feature-default annotation, using default componentPolicy for upgrades",
		"unexpectedComponentPolicy", recordedValue,
		"componentPolicy", defaultForUpgrades)
	return securedClusterScannerDefaultForUpgrades, true
}
