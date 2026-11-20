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
	// IncludeResults indicates whether the cluster's check results should stay in the
	// report content even though the cluster is flagged. It is set when the findings are
	// staleness (overdue scans) rather than execution failures: the latest results per
	// scan are still the most complete data available, and mixed freshness is the
	// intended semantics of settle-audit-snapshot reporting. When false and
	// FailedScans is empty, the whole cluster is excluded from the report content.
	IncludeResults bool
}
