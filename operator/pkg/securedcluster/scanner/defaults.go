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

// SetScannerV4Defaults makes sure that spec.ScannerV4 and spec.ScannerV4.ScannerComponent are not nil.
func SetScannerV4Defaults(spec *platform.SecuredClusterSpec) {
	if spec.ScannerV4 == nil {
		spec.ScannerV4 = &platform.LocalScannerV4ComponentSpec{}
	}
	if spec.ScannerV4.ScannerComponent == nil {
		spec.ScannerV4.ScannerComponent = platform.LocalScannerComponentAutoSense.Pointer()
	}
}
