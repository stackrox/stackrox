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
	if request.ReportConfig == nil {
		errorList.AddError(errors.New("Report request does not have a valid non-nil report configuration"))
	} else if request.ReportConfig.GetId() == "" {
		errorList.AddError(errors.New("Report configuration ID is empty"))
	}
	if request.Collection == nil {
		errorList.AddError(errors.New("Report request does not have a valid non-nil collection."))
	}

	if request.ReportSnapshot == nil {
		errorList.AddError(errors.New("Report request does not have a valid report snapshot with report status"))
	} else {
		if request.ReportSnapshot.ReportStatus == nil {
			errorList.AddError(errors.New("Report request does not have a valid report snapshot with report status"))
		}
		if request.ReportSnapshot.GetReportId() == "" {
			errorList.AddError(errors.New("Report ID is empty"))
		}
		if request.ReportConfig.GetId() != request.ReportSnapshot.GetReportConfigurationId() {
			errorList.AddError(errors.New("Mismatch between report config ids in ReportConfig and ReportSnapshot"))
		}
	}
	return errorList.ToError()
}
