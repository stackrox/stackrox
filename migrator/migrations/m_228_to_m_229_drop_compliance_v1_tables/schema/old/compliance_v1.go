// Frozen copy of the generated schemas for the five Compliance 1.0 tables, taken at
// the time those tables were removed (ROX-36856 Compliance 1.0 removal). Kept here
// so this migration can reference table names and create statements without depending
// on the live schema package. DO NOT EDIT.

package old

import (
	"time"

	"github.com/stackrox/rox/pkg/postgres"
)

const (
	ComplianceConfigsTableName     = "compliance_configs"
	ComplianceDomainsTableName     = "compliance_domains"
	ComplianceStringsTableName     = "compliance_strings"
	ComplianceRunResultsTableName  = "compliance_run_results"
	ComplianceRunMetadataTableName = "compliance_run_metadata"
)

var (
	CreateTableComplianceConfigsStmt = &postgres.CreateStmts{
		GormModel: (*ComplianceConfigs)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	CreateTableComplianceDomainsStmt = &postgres.CreateStmts{
		GormModel: (*ComplianceDomains)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	CreateTableComplianceStringsStmt = &postgres.CreateStmts{
		GormModel: (*ComplianceStrings)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	CreateTableComplianceRunResultsStmt = &postgres.CreateStmts{
		GormModel: (*ComplianceRunResults)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	CreateTableComplianceRunMetadataStmt = &postgres.CreateStmts{
		GormModel: (*ComplianceRunMetadata)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	CreateTableComplianceIntegrationsStmt = &postgres.CreateStmts{
		GormModel: (*ComplianceIntegrations)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

type ComplianceConfigs struct {
	StandardID string `gorm:"column:standardid;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}

type ComplianceDomains struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}

type ComplianceStrings struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}

type ComplianceRunResults struct {
	RunMetadataRunID           string     `gorm:"column:runmetadata_runid;type:varchar;primaryKey"`
	RunMetadataStandardID      string     `gorm:"column:runmetadata_standardid;type:varchar"`
	RunMetadataClusterID       string     `gorm:"column:runmetadata_clusterid;type:uuid"`
	RunMetadataFinishTimestamp *time.Time `gorm:"column:runmetadata_finishtimestamp;type:timestamp"`
	Serialized                 []byte     `gorm:"column:serialized;type:bytea"`
}

type ComplianceRunMetadata struct {
	RunID           string     `gorm:"column:runid;type:varchar;primaryKey"`
	StandardID      string     `gorm:"column:standardid;type:varchar"`
	ClusterID       string     `gorm:"column:clusterid;type:uuid"`
	FinishTimestamp *time.Time `gorm:"column:finishtimestamp;type:timestamp"`
	Serialized      []byte     `gorm:"column:serialized;type:bytea"`
}

type ComplianceIntegrations struct {
	ID                string `gorm:"column:id;type:uuid;primaryKey"`
	Version           string `gorm:"column:version;type:varchar"`
	ClusterID         string `gorm:"column:clusterid;type:uuid;uniqueIndex:compliance_unique_indicator"`
	OperatorInstalled bool   `gorm:"column:operatorinstalled;type:bool"`
	OperatorStatus    int32  `gorm:"column:operatorstatus;type:integer"`
	Serialized        []byte `gorm:"column:serialized;type:bytea"`
}
