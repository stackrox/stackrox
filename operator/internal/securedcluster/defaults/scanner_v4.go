package defaults

import (
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
// Scanner V4 is now always on (scanner v2 has been removed), so we default to
// AutoSense unless the user explicitly set Disabled in the CR spec.
//
// Second return value is `true`, if defaulting has been applied due to lack of explicit setting.
func SecuredClusterScannerV4ComponentPolicy(logger logr.Logger, _ *platform.SecuredClusterStatus, _ map[string]string, spec *platform.LocalScannerV4ComponentSpec) (platform.LocalScannerV4ComponentPolicy, bool) {
	if spec != nil && spec.ScannerComponent != nil {
		comp := *spec.ScannerComponent
		if comp == platform.LocalScannerV4AutoSense || comp == platform.LocalScannerV4Disabled {
			logger.Info("using componentPolicy set in CR", "componentPolicy", comp)
			return comp, false
		}
	}

	return platform.LocalScannerV4AutoSense, true
}

func securedClusterScannerV4Defaulting(logger logr.Logger, status *platform.SecuredClusterStatus, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
	scannerV4Spec := copyLocalScannerV4ComponentSpec(spec.ScannerV4)
	componentPolicy, usedDefaulting := SecuredClusterScannerV4ComponentPolicy(logger, status, annotations, scannerV4Spec)
	if !usedDefaulting {
		// User provided an explicit choice, nothing to do in this flow.
		return nil
	}

	// User is relying on defaults. Set in-memory default only.
	// We no longer persist the annotation to support downgrade scenarios.

	if defaults.ScannerV4 == nil {
		defaults.ScannerV4 = &platform.LocalScannerV4ComponentSpec{}
	}
	defaults.ScannerV4.ScannerComponent = &componentPolicy
	return nil
}

func copyLocalScannerV4ComponentSpec(spec *platform.LocalScannerV4ComponentSpec) *platform.LocalScannerV4ComponentSpec {
	if spec == nil {
		return &platform.LocalScannerV4ComponentSpec{}
	}
	return spec.DeepCopy()
}
