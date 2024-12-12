package common

import (
	"fmt"
)

const (
	ReportBlobPathPrefix   = "/central/reports/"
	reportBlobPathTemplate = ReportBlobPathPrefix + "%s/%s"

	// ReportBlobRegex matches all downloadable report blob names
	ReportBlobRegex = "^(" + ReportBlobPathPrefix + "|" + ComplianceReportBlobPathPrefix + ").+"

	ComplianceReportBlobPathPrefix   = "/central/compliance/reports/"
	complianceReportBlobPathTemplate = ComplianceReportBlobPathPrefix + "%s/%s"
)

// GetReportBlobPath creates the Blob path for report
func GetReportBlobPath(configID, reportID string) string {
	return fmt.Sprintf(reportBlobPathTemplate, configID, reportID)
}

// GetComplianceReportBlobPath creates the Blob path for the compliance report
func GetComplianceReportBlobPath(configID, reportID string) string {
	return fmt.Sprintf(complianceReportBlobPathTemplate, configID, reportID)
}
