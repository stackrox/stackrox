package defaulting

import (
	"reflect"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

var (
	SecuredClusterScannerV4DefaultingFlow = SecuredClusterDefaultingFlow{
		Name:           "scanner-V4",
		DefaultingFunc: securedClusterScannerV4Defaulting,
	}
)

// Only returns AutoSense or Disabled.
// Derive component policy based on status Defaults and spec.
// This will be called from the preExtension to record the current setting.
//
// Second return value is `true`, if defaulting has been applied due to lack of explicit setting.
func SecuredClusterScannerV4ComponentPolicy(logger logr.Logger, status *platform.SecuredClusterStatus, annotations map[string]string, spec *platform.LocalScannerV4ComponentSpec) (platform.LocalScannerV4ComponentPolicy, bool) {
	defaultForUpgrades := platform.LocalScannerV4Disabled
	defaultForNewInstallations := platform.LocalScannerV4AutoSense
	logger = logger.WithName("scanner-v4-defaulting")

	if spec != nil && spec.ScannerComponent != nil {
		comp := *spec.ScannerComponent
		if comp == platform.LocalScannerV4AutoSense || comp == platform.LocalScannerV4Disabled {
			logger.Info("using componentPolicy set in CR", "componentPolicy", comp)
			return comp, false
		}
	}

	// User is relying on defaulting.
	// This includes the case spec.ScannerComponent == "Default".

	// A default entry exists already, use recorded value.
	recordedValue := platform.LocalScannerV4ComponentPolicy(annotations[FeatureDefaultKeyScannerV4])
	if recordedValue == platform.LocalScannerV4AutoSense || recordedValue == platform.LocalScannerV4Disabled {
		logger.Info("using previously recorded componentPolicy", "componentPolicy", recordedValue)
		return recordedValue, true
	}

	// No or unexpected default set in the annotations.

	if securedClusterStatusUninitialized(status) {
		// Install / Green field.
		logger.Info("assuming new installation due to empty status.")
		return defaultForNewInstallations, true
	}

	// Upgrade.
	logger.Info("assuming upgrade")

	if annotations[FeatureDefaultKeyScannerV4] == "" {
		// No entry in the statusDefaults yet -> preserve defaulting behavior of versions which did not populate
		// statusDefaults with a ScannerV4ComponentPolicy.
		logger.Info("empty feature-default annotation, using default componentPolicy for upgrades",
			"componentPolicy", defaultForUpgrades)
		return defaultForUpgrades, true
	}

	// This should not happen, since we only store |Disabled, but just in case something unexpected happened
	// and we need to make some decisions...
	logger.Info("detected unexpected componentPolicy in feature-default annotation, using default componentPolicy for upgrades",
		"unexpectedComponentPolicy", recordedValue,
		"componentPolicy", defaultForUpgrades)
	return defaultForUpgrades, true
}

// securedClusterStatusUninitialized checks if the provided Securedcluster status is uninitialized.
func securedClusterStatusUninitialized(status *platform.SecuredClusterStatus) bool {
	return status == nil || reflect.DeepEqual(status, &platform.SecuredClusterStatus{})
}

func securedClusterScannerV4Defaulting(logger logr.Logger, status *platform.SecuredClusterStatus, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
	scannerV4Spec := copyLocalScannerV4ComponentSpec(spec.ScannerV4)
	componentPolicy, usedDefaulting := SecuredClusterScannerV4ComponentPolicy(logger, status, annotations, scannerV4Spec)
	if !usedDefaulting {
		// User provided an explicit choice, nothing to do in this flow.
		return nil
	}

	// User is relying on defaults. Set in-memory default and persist corresponding annotation.

	if annotations[FeatureDefaultKeyScannerV4] != string(componentPolicy) {
		// Update feature default setting.
		annotations[FeatureDefaultKeyScannerV4] = string(componentPolicy)
	}

	defaults.ScannerV4 = &platform.LocalScannerV4ComponentSpec{ScannerComponent: &componentPolicy}
	return nil
}

func copyLocalScannerV4ComponentSpec(spec *platform.LocalScannerV4ComponentSpec) *platform.LocalScannerV4ComponentSpec {
	if spec == nil {
		return &platform.LocalScannerV4ComponentSpec{}
	}
	return spec.DeepCopy()
}
