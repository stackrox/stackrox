package npg

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/roxctl/common/logger"
)

var (
	// ErrErrors is a marker-error message that marks occurrence of errors
	ErrErrors = errors.New("there were errors during execution")
	// ErrWarnings is a marker-error message that marks occurrence of warnings
	ErrWarnings = errors.New("there were warnings during execution")
)

// NPGuardErrorType summarizes commonalities of three types of errors returned by NP-Guard
type NPGuardErrorType interface {
	Error() error
	Location() string
	IsSevere() bool
}

// HandleNPGuardErrors classifies NP-Guard errors as warnings or errors and ensures
// that error-related location is included in the error message
func HandleNPGuardErrors[T NPGuardErrorType](src []T) (warns []error, errs []error) {
	for _, err := range src {
		e := err.Error()
		if err.Location() != "" {
			e = errors.Errorf("%s (at %q)", err.Error(), err.Location())
		}
		if err.IsSevere() {
			errs = append(errs, e)
		} else {
			warns = append(warns, e)
		}
	}
	return warns, errs
}

// SummarizeErrors returns appropriate error-marker if the operation should be considered as erroneous.
// It displays errors and warnings using the provided logger
func SummarizeErrors(warns []error, errs []error, treatWarningsAsErrors bool, logger logger.Logger) error {
	var errToReturn error
	if len(errs) > 0 {
		errToReturn = ErrErrors
	} else if treatWarningsAsErrors && len(warns) > 0 {
		errToReturn = ErrWarnings
	}
	for _, warn := range warns {
		logger.WarnfLn("%s", warn.Error())
	}
	for _, err := range errs {
		logger.ErrfLn("%s", err.Error())
	}
	return errToReturn
}
