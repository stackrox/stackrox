package defaults

import (
	"github.com/go-logr/logr"
	"github.com/stackrox/rox/operator/api/v1alpha1"
)

// CentralDefaultingFlow defines a defaulting flow for the Central CR.
// Any mutation to either `spec` or `status` is not preserved.
type CentralDefaultingFlow struct {
	Name           string
	DefaultingFunc func(logger logr.Logger, status *v1alpha1.CentralStatus, annotations map[string]string, spec *v1alpha1.CentralSpec, defaults *v1alpha1.CentralSpec) error
}
