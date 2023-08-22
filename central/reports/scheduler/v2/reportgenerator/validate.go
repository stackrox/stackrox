package reportgenerator

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// ValidateReportRequest validates the report request. It performs some basic nil checks, empty checks
// and checks if report configuration ID is same in both report configuration and report metadata.
// These are basic sanity checks and not checking user errors.
func ValidateReportRequest(request *ReportRequest) error {
	if request == nil {
		return errors.New("Report request is nil.")
	}
	errorList := errorhelpers.NewErrorList("validating report request")
	if request.Collection == nil {
		errorList.AddError(errors.New("Report request does not have a valid non-nil collection."))
	}

	if request.ReportSnapshot == nil {
		errorList.AddError(errors.New("Report request does not have a valid report snapshot with report status"))
	} else if request.ReportSnapshot.ReportStatus == nil {
		errorList.AddError(errors.New("Report request does not have a valid report snapshot with report status"))
	}
	return errorList.ToError()
}
