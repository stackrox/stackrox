package defaults

import (
	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

var (
	CentralScannerV4DefaultingFlow = CentralDefaultingFlow{
		Name:           "scanner-V4",
		DefaultingFunc: centralScannerV4Defaulting,
	}
)

// Only returns Enabled or Disabled.
// Scanner V4 is now always on (scanner v2 has been removed), so we default to
// Enabled unless the user explicitly set Disabled in the CR spec.
//
// Second return value is `true`, if defaulting has been applied due to lack of explicit setting.
func CentralScannerV4ComponentPolicy(logger logr.Logger, _ *platform.CentralStatus, _ map[string]string, spec *platform.ScannerV4Spec) (platform.ScannerV4ComponentPolicy, bool) {
	if spec != nil && spec.ScannerComponent != nil {
		comp := *spec.ScannerComponent
		if comp == platform.ScannerV4ComponentEnabled || comp == platform.ScannerV4ComponentDisabled {
			logger.Info("using ScannerV4 componentPolicy set in CR", "componentPolicy", comp)
			return comp, false
		}
	}

	return platform.ScannerV4Enabled, true
}

func centralScannerV4Defaulting(logger logr.Logger, status *platform.CentralStatus, annotations map[string]string, spec *platform.CentralSpec, defaults *platform.CentralSpec) error {
	scannerV4Spec := copyScannerV4Spec(spec.ScannerV4)
	componentPolicy, usedDefaulting := CentralScannerV4ComponentPolicy(logger, status, annotations, scannerV4Spec)
	if !usedDefaulting {
		// User provided an explicit choice, nothing to do in this flow.
		return nil
	}

	// User is relying on defaults. Set in-memory default only.
	// We no longer persist the annotation to support downgrade scenarios.

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
