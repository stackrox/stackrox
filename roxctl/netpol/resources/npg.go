package resources

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/roxctl/common/npg"
)

type ErrorLocationSeverity interface {
	Error() error
	Location() string
	IsSevere() bool
}

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
