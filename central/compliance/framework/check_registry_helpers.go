package framework

import (
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/utils"
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

// MustRegisterNewCheckIfFlagEnabled calls MustRegisterNewCheck if the given feature flag is enabled.
func MustRegisterNewCheckIfFlagEnabled(metadata CheckMetadata, checkFn CheckFunc, flag features.FeatureFlag) {
	if !flag.Enabled() {
		return
	}
	MustRegisterNewCheck(metadata, checkFn)
}
