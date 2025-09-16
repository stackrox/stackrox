package utils

import (
	"fmt"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils/panic"
)

// Must panics if any of the given errors is non-nil, and does nothing otherwise.
func Must(errs ...error) {
	CrashOnError(errs...)
}

// CrashOnError is an alternative to `Must`. It was introduced because `Must(err)` looks confusing.
func CrashOnError(errs ...error) {
	for _, err := range errs {
		if err != nil {
			panic.HardPanic(fmt.Sprintf("%+v", err))
		}
	}
}

// ShouldErr panics on development builds and logs on release builds
// The expectation is that this function will be called with an error wrapped by errors.Wrap
// so that tracing is easier
func ShouldErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			if buildinfo.ReleaseBuild {
				logging.Errorf("Unexpected Error: %+v", err)
			} else {
				panic.HardPanic(err)
			}
			return err
		}
	}
	return nil
}

// Should wraps ShouldErr without returning the error. This removes gosec G104.
func Should(errs ...error) {
	_ = ShouldErr(errs...)
}
