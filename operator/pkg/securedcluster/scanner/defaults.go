package scanner

import platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"

// SetScannerDefaults makes sure that spec.Scanner and spec.Scanner.ScannerComponent are not nil.
func SetScannerDefaults(spec *platform.SecuredClusterSpec) {
	if spec.Scanner == nil {
		spec.Scanner = &platform.LocalScannerComponentSpec{}
	}
	if spec.Scanner.ScannerComponent == nil {
		spec.Scanner.ScannerComponent = platform.LocalScannerComponentAutoSense.Pointer()
	}
}
