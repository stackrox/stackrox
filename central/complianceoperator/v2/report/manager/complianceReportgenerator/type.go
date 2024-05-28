package complianceReportgenerator

import (
	"github.com/stackrox/rox/generated/storage"
)

type ComplianceReportRequest struct {
	scanConfigID   string
	notifiers      []*storage.NotifierConfiguration
	clusterIDs     []string
	profiles       []string
	scanConfigName string
}
