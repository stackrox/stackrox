package complianceReportgenerator

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

type ComplianceReportRequest struct {
	ScanConfigID       string
	Notifiers          []*storage.NotifierConfiguration
	ClusterIDs         []string
	Profiles           []string
	ScanConfigName     string
	Ctx                context.Context
	SnapshotID         string
	NotificationMethod storage.ComplianceOperatorReportStatus_NotificationMethod
}
