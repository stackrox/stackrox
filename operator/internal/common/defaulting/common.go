package defaulting

import (
	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

func initializedDeepCopy(spec *platform.ScannerV4Spec) *platform.ScannerV4Spec {
	if spec == nil {
		return &platform.ScannerV4Spec{}
	}
	return spec.DeepCopy()
}

// CentralDefaultingFlow defines a defaulting flow for the Central CR.
// Any mutation to either `spec` or `status` is not preserved.
type CentralDefaultingFlow struct {
	Name           string
	DefaultingFunc func(logger logr.Logger, status *platform.CentralStatus, annotations map[string]string, spec *platform.CentralSpec, defaults *platform.CentralSpec) error
}
