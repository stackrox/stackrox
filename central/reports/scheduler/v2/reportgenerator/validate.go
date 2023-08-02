package reportgenerator

import (
	"github.com/pkg/errors"
)

// ValidateReportRequest validates the report request. It performs some basic nil checks, empty checks
// and checks if report configuration ID is same in both report configuration and report metadata.
func ValidateReportRequest(request *ReportRequest) error {
	if request == nil {
		return errors.New("Report request is nil.")
	}

	if request.ReportConfig == nil {
		return errors.New("Report request does not have a valid non-nil report configuration")
	}

	if request.ReportConfig.GetId() == "" {
		return errors.New("Report configuration ID is empty")
	}

	if request.Collection == nil {
		return errors.New("Report request does not have a valid non-nil collection.")
	}

	if request.ReportSnapshot.GetReportId() == "" {
		return errors.New("Report ID is empty")
	}

	if request.ReportSnapshot == nil || request.ReportSnapshot.ReportStatus == nil {
		return errors.New("Report request does not have a valid report snapshot with report status")
	}

	if request.ReportConfig.GetId() != request.ReportSnapshot.GetReportConfigurationId() {
		return errors.New("Mismatch between report config ids in ReportConfig and ReportSnapshot")
	}

	return nil
}
