package framework

import (
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/utils"
)

// RegisterChecks registers a check in the global registry.
func RegisterChecks(checks ...Check) error {
	errList := errorhelpers.NewErrorList("registering checks")
	registry := RegistrySingleton()
	for _, check := range checks {
		if err := registry.Register(check); err != nil {
			errList.AddError(err)
		}
	}
	return errList.ToError()
}

// MustRegisterChecks registers a check in the global registry, and panics if the check could not be registered.
func MustRegisterChecks(checks ...Check) {
	utils.Must(RegisterChecks(checks...))
}

// MustRegisterNewCheck creates a check from a function with the given metadata and registers it. If an error occurs,
// it panics.
func MustRegisterNewCheck(metadata CheckMetadata, checkFn CheckFunc) {
	MustRegisterChecks(NewCheckFromFunc(metadata, checkFn))
}
