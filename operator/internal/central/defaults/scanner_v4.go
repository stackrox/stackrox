package defaults

import (
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
// Derive component policy based on spec.
// This will be called from the preExtension to record the current setting.
//
// Second return value is `true`, if defaulting has been applied due to lack of explicit setting.
//
// Scanner V2 is retired, so Scanner V4 is now enabled by default for both new installations
// and upgrades. The only way to keep Scanner V4 off is an explicit CR "Disabled" (a user who
// accepts running with no scanner). We deliberately no longer honor a recorded/stale
// "Disabled" feature-default annotation nor apply the previous pre-4.8 upgrade exception:
// installs that relied on defaulting migrate to Scanner V4. See the package plan notes and
// centralScannerV4Defaulting, which overwrites the annotation to the effective (Enabled)
// value so the recorded pin stays truthful (and downgrade to N-1 keeps V4 on).
func CentralScannerV4ComponentPolicy(logger logr.Logger, status *platform.CentralStatus, annotations map[string]string, spec *platform.ScannerV4Spec) (platform.ScannerV4ComponentPolicy, bool) {
	if spec != nil && spec.ScannerComponent != nil {
		comp := *spec.ScannerComponent
		if comp == platform.ScannerV4ComponentEnabled || comp == platform.ScannerV4ComponentDisabled {
			logger.Info("using ScannerV4 componentPolicy set in CR", "componentPolicy", comp)
			return comp, false
		}
	}

	// User is relying on defaulting (this includes the case spec.ScannerComponent == "Default").
	// Scanner V4 is enabled by default in all such cases.
	logger.Info("using default ScannerV4 componentPolicy", "componentPolicy", platform.ScannerV4Enabled)
	return platform.ScannerV4Enabled, true
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
