package resources

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/roxctl/common/npg"
)

// ErrorLocationSeverity is an error that can be severe and has a reference to a location (e.g., file on disk)
type ErrorLocationSeverity interface {
	Error() error
	Location() string
	IsSevere() bool
}

// HandleNPGerrors classifies NP-Guard errors as warnings or errors
func HandleNPGerrors(src []ErrorLocationSeverity, treatWarningsAsErrors bool) (warns []error, errs []error) {
	var roxerr error
	for _, err := range src {
		if err.IsSevere() {
			errs = append(errs, errors.Wrap(err.Error(), err.Location()))
			roxerr = npg.ErrErrors
		} else {
			warns = append(warns, errors.Wrap(err.Error(), err.Location()))
			if treatWarningsAsErrors && roxerr == nil {
				roxerr = npg.ErrWarnings
			}
		}
	}
	if roxerr != nil {
		errs = append(errs, roxerr)
	}
	return warns, errs
}
