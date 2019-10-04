package utils

import (
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/logging"
)

// Must panics if any of the given errors is non-nil, and does nothing otherwise.
func Must(errs ...error) {
	for _, err := range errs {
		if err != nil {
			panic(err)
		}
	}
}

// Should panics on development builds and logs on release builds
// The expectation is that this function will be called with an error wrapped by errors.Wrap
// so that tracing is easier
func Should(errs ...error) {
	for _, err := range errs {
		if err != nil {
			if buildinfo.ReleaseBuild {
				logging.Errorf("Unexpected Error: %+v", err)
			} else {
				panic(err)
			}
		}
	}
}
