package defaulting

import (
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	operatorVersion "github.com/stackrox/rox/operator/internal/version"
	"github.com/stackrox/rox/pkg/version"
)

var (
	defaultForNewInstallations = platform.ScannerV4Enabled
	ownerVersion               = version.MustParseXYVersion("4.8")
)

// Only returns Enabled or Disabled.
// Derive component policy based on status Defaults and spec.
// This will be called from the preExtension to record the current setting.
func ScannerV4ComponentPolicy(statusDefaults *platform.StatusDefaults, spec *platform.ScannerV4Spec) platform.ScannerV4ComponentPolicy {
	if spec != nil && spec.ScannerComponent != nil {
		if *spec.ScannerComponent == platform.ScannerV4ComponentEnabled ||
			*spec.ScannerComponent == platform.ScannerV4ComponentDisabled {
			return *spec.ScannerComponent
		}
	}

	// User is relying on defaulting.
	// This includes the case spec.ScannerComponent == "Default".

	if statusDefaults == nil {
		// Install.
		return defaultForNewInstallations
	}

	// Upgrade.

	statusDefault := statusDefaults.ScannerV4ComponentPolicy
	if statusDefault == (platform.StatusDefault{}) {
		// No entry in the statusDefaults yet -> preserve defaulting behavior of versions which did not populate
		// statusDefaults with a ScannerV4ComponentPolicy.
		return platform.ScannerV4Disabled
	}

	// A statusDefault entry exists already.

	ownerVersion, err := version.ParseXYVersion(statusDefault.OwnerVersion)
	if err != nil {
		// Swallow error.
		return defaultForNewInstallations
	}

	if operatorVersion.XYVersion.LessOrEqual(ownerVersion) {
		// Current operator version is <= recorded version.
		// Which means we could be on the same major.minor version or a downgrade has occurred.
		// In any case, do not use the recorded value.
		return defaultForNewInstallations
	}

	// Recorded default comes from a previous XY version, use recorded value.
	recordedComponentPolicy := platform.ScannerV4ComponentPolicy(statusDefault.Value)
	if recordedComponentPolicy == platform.ScannerV4Enabled || recordedComponentPolicy == platform.ScannerV4Disabled {
		return recordedComponentPolicy
	} else {
		// This should not happen, since we only store Enabled|Disabled, but just in case something unexpected happened
		// and we need to make some decisions...
		return defaultForNewInstallations
	}
}

// Convenience for some callers.
func ScannerV4ComponentPolicyEnabled(statusDefaults *platform.StatusDefaults, spec *platform.ScannerV4Spec) bool {
	componentPolicy := ScannerV4ComponentPolicy(statusDefaults, spec)
	return componentPolicy == platform.ScannerV4ComponentEnabled
}

// Mutating version of the above function. Updates a given spec by filling in the computed setting.
// This will be called during value translations.
func ScannerV4DefaultsApply(statusDefaults *platform.StatusDefaults, spec *platform.ScannerV4Spec) {
	// assert spec != nil
	val := ScannerV4ComponentPolicy(statusDefaults, spec)
	spec.ScannerComponent = &val
}
