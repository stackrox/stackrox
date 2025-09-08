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

	if request.ReportSnapshot == nil {
		errorList.AddError(errors.New("Report request does not have a valid report snapshot with report status"))
	} else if request.ReportSnapshot.ReportStatus == nil {
		errorList.AddError(errors.New("Report request does not have a valid report snapshot with report status"))
	}
	//only check collection is non nil if report snapshot is for config based vuln reports
	if request.ReportSnapshot.GetVulnReportFilters() != nil && request.Collection == nil {
		errorList.AddError(errors.New("Report request does not have a valid non-nil collection."))
	}

	return errorList.ToError()
}
