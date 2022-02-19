package extensions

import (
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
)

const (
	testNamespace = `testns`
)

func basicSpecWithScanner(scannerEnabled bool) platform.CentralSpec {
	spec := platform.CentralSpec{
		Scanner: &platform.ScannerComponentSpec{
			ScannerComponent: new(platform.ScannerComponentPolicy),
		},
	}
	if scannerEnabled {
		*spec.Scanner.ScannerComponent = platform.ScannerComponentEnabled
	} else {
		*spec.Scanner.ScannerComponent = platform.ScannerComponentDisabled
	}
	return spec
}
