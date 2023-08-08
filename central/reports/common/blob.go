package common

import (
	"fmt"
)

const (
	reportBlobPathPrefix   = "/central/reports/"
	reportBlobPathTemplate = reportBlobPathPrefix + "%s/%s"
)

// GetReportBlobPath creates the Blob path for report
func GetReportBlobPath(configID, reportID string) string {
	return fmt.Sprintf(reportBlobPathTemplate, configID, reportID)
}
