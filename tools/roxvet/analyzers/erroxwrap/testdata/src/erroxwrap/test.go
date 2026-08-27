package erroxwrap

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
)

func badWrap() error {
	return errors.Wrap(errox.InvalidArgs, "bad wrap") // want "Use errox.InvalidArgs.New\\(\\.\\.\\.\\) to create an error with a message"
}

func badWrapf() error {
	return errors.Wrapf(errox.NotFound, "bad wrap %s", "test") // want "Use errox.NotFound.Newf\\(\\.\\.\\.\\) to create an error with a formatted message"
}

func badWrapMultiple() error {
	err1 := errors.Wrap(errox.AlreadyExists, "first bad wrap") // want "Use errox.AlreadyExists.New\\(\\.\\.\\.\\) to create an error with a message"
	err2 := errors.Wrap(errox.InvalidArgs, "second bad wrap")  // want "Use errox.InvalidArgs.New\\(\\.\\.\\.\\) to create an error with a message"
	if err1 != nil {
		return err1
	}
	return err2
}

func badWrapWithError() error {
	err := someFunction()
	return errors.Wrapf(errox.InvalidArgs, "failed to process: %v", err) // want "Use errox.InvalidArgs.CausedByf\\(\\.\\.\\.\\) to wrap an error with a formatted message"
}

// These should be fine - not flagged
func goodNew() error {
	return errox.InvalidArgs.New("good usage")
}

func goodNewf() error {
	return errox.NotFound.Newf("good usage %s", "test")
}

func goodCausedBy() error {
	err := someFunction()
	return errox.InvalidArgs.CausedBy(err)
}

func goodCausedByf() error {
	err := someFunction()
	return errox.InvalidArgs.CausedByf("failed: %v", err)
}

// Wrapping non-errox errors should be fine
func goodWrapNonErrox() error {
	err := someFunction()
	return errors.Wrap(err, "wrapping regular error is fine")
}

func someFunction() error {
	return nil
}
