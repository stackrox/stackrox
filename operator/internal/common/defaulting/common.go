package defaulting

import (
	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

type CentralDefaultingFlow struct {
	Name           string
	DefaultingFunc func(logger logr.Logger, status *platform.CentralStatus, annotations map[string]string, spec *platform.CentralSpec, defaults *platform.CentralSpec) error
}

type SecuredClusterDefaultingFlow struct {
	Name           string
	DefaultingFunc func(logger logr.Logger, status *platform.SecuredClusterStatus, annotations map[string]string, spec *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error
}
