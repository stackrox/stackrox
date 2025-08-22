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
	ClusterData        map[string]*ClusterData
	NumFailedClusters  int
}

// ClusterData holds the metadata for the clusters
type ClusterData struct {
	ClusterId   string
	ClusterName string
	ScanNames   []string
	FailedInfo  *FailedCluster
}

// FailedCluster holds the information of a failed cluster
type FailedCluster struct {
	ClusterId       string
	ClusterName     string
	Reasons         []string
	OperatorVersion string
	FailedScans     []*storage.ComplianceOperatorScanV2
}
