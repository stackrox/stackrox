package defaults

import (
	"github.com/go-logr/logr"
	"github.com/stackrox/rox/operator/api/v1alpha1"
)

// SecuredClusterDefaultingFlow defines a defaulting flow for the SecuredCluster CR.
// Any mutation to either `spec` or `status` is not preserved.
type SecuredClusterDefaultingFlow struct {
	Name           string
	DefaultingFunc func(logger logr.Logger, status *v1alpha1.SecuredClusterStatus, annotations map[string]string, spec *v1alpha1.SecuredClusterSpec, defaults *v1alpha1.SecuredClusterSpec) error
}
