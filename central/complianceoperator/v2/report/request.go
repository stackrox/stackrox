package report

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Request holds the input data for a compliance report request
type Request struct {
	ScanConfigID       string
	Notifiers          []*storage.NotifierConfiguration
	ClusterIDs         []string
	Profiles           []string
	ScanConfigName     string
	Ctx                context.Context
	SnapshotID         string
	NotificationMethod storage.ComplianceOperatorReportStatus_NotificationMethod
	FailedClusters     map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster
}
