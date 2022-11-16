package utils

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	hardPanicDelay = 5 * time.Second
)

// hardPanic is like panic, but on debug builds additionally ensures that the panic will cause a crash with a full
// goroutine dump, independently of any recovery handlers.
func hardPanic(v interface{}) {
	if !buildinfo.ReleaseBuild {
		trace := debug.GetLazyStacktrace(2)
		time.AfterFunc(hardPanicDelay, func() {
			panic(fmt.Sprintf("Re-triggering panic %v as unrecoverable. Original stacktrace:\n%s", v, trace))
		})
	}
	panic(v)
}

// Must panics if any of the given errors is non-nil, and does nothing otherwise.
func Must(errs ...error) {
	CrashOnError(errs...)
}

// CrashOnError is an alternative to `Must`. It was introduced because `Must(err)` looks confusing.
func CrashOnError(errs ...error) {
	for _, err := range errs {
		if err != nil {
			hardPanic(fmt.Sprintf("%+v", err))
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
				hardPanic(err)
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
