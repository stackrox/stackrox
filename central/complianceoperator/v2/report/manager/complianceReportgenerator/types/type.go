package types

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

// ResultRow struct which hold all columns of a report row
type ResultRow struct {
	ClusterName  string
	CheckName    string
	Profile      string
	ControlRef   string
	Description  string
	Status       string
	Remediation  string
	Rationale    string
	Instructions string
}
