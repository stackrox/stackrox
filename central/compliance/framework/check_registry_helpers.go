package framework

import (
	"errors"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/utils"
)

// RegisterChecks registers a check in the global registry.
func RegisterChecks(checks ...Check) error {
	var registerChecksErrs error
	registry := RegistrySingleton()
	for _, check := range checks {
		if err := registry.Register(check); err != nil {
			registerChecksErrs = errors.Join(registerChecksErrs, err)
		}
	}
	return pkgErrors.Wrap(registerChecksErrs, "checking registry")
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
