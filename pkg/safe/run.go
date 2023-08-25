package safe

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// RunE executes the given function, and will wrap any panic that it encountered as an error. Any error returned
// from fn() in normal execution is passed through as-is.
// Note: the error is passed through `utils.Should`, resulting in a panic on debug builds and a log message on
// release builds.
func RunE(fn func() error) error {
	return utils.ShouldErr(runE(fn))
}

// runE is like RunE, but the result is not passed through utils.Should for better testability.
func runE(fn func() error) (err error) {
	panicked := true
	defer func() {
		if !panicked {
			return
		}
		r := recover()
		rErr, _ := r.(error)
		if rErr == nil {
			rErr = errors.Errorf("recovered: %v", r)
		}
		err = errors.Wrap(rErr, "caught panic")
	}()

	err = fn()
	panicked = false
	return
}

// Run executes the given function, and will wrap any panic that it encountered as an error.
// This function returns nil if and only if fn() did not cause a panic.
func Run(fn func()) error {
	return RunE(func() error {
		fn()
		return nil
	})
}
