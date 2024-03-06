package reportgenerator

import (
	"errors"
	"fmt"

	"github.com/stackrox/rox/pkg/errox"
)

// ValidateReportRequest validates the report request. It performs some basic nil checks, empty checks
// and checks if report configuration ID is same in both report configuration and report metadata.
// These are basic sanity checks and not checking user errors.
func ValidateReportRequest(request *ReportRequest) error {
	if request == nil {
		return errors.New("Report request is nil.")
	}
	var validateErrs error
	if request.Collection == nil {
		validateErrs = errors.Join(validateErrs,
			errox.InvalidArgs.New("report request does not have a valid non-nil collection"))
	}

	if request.ReportSnapshot == nil {
		validateErrs = errors.Join(validateErrs,
			errox.InvalidArgs.New("report request does not have a valid report snapshot with report status"))
	} else if request.ReportSnapshot.ReportStatus == nil {
		validateErrs = errors.Join(validateErrs,
			errox.InvalidArgs.New("report request does not have a valid report snapshot with report status"))
	}
	if validateErrs != nil {
		return fmt.Errorf("validating report request: %w", validateErrs)
	}
	return nil
}
