package common

import (
	"fmt"
)

const (
	reportBlobPathPrefix   = "/central/reports/"
	reportBlobPathTemplate = reportBlobPathPrefix + "%s/%s"

	// ReportBlobRegex matches all downloadable report blob names
	ReportBlobRegex = "^" + reportBlobPathPrefix + ".+"
)

// GetReportBlobPath creates the Blob path for report
func GetReportBlobPath(configID, reportID string) string {
	return fmt.Sprintf(reportBlobPathTemplate, configID, reportID)
}
