// Frozen copy of the generated schemas for the five Compliance Operator v1 tables, taken at
// the time those tables were removed (ROX-36856 commit 754c03f3bd). Kept here so this
// migration can reference table names and create statements without depending on the live
// schema package. DO NOT EDIT.

package old

import (
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	ComplianceOperatorCheckResultsTableName        = "compliance_operator_check_results"
	ComplianceOperatorProfilesTableName            = "compliance_operator_profiles"
	ComplianceOperatorRulesTableName               = "compliance_operator_rules"
	ComplianceOperatorScanSettingBindingsTableName = "compliance_operator_scan_setting_bindings"
	ComplianceOperatorScansTableName               = "compliance_operator_scans"
)

var (
	CreateTableComplianceOperatorCheckResultsStmt        = &postgres.CreateStmts{GormModel: (*ComplianceOperatorCheckResults)(nil), Children: []*postgres.CreateStmts{}}
	CreateTableComplianceOperatorProfilesStmt            = &postgres.CreateStmts{GormModel: (*ComplianceOperatorProfiles)(nil), Children: []*postgres.CreateStmts{}}
	CreateTableComplianceOperatorRulesStmt               = &postgres.CreateStmts{GormModel: (*ComplianceOperatorRules)(nil), Children: []*postgres.CreateStmts{}}
	CreateTableComplianceOperatorScanSettingBindingsStmt = &postgres.CreateStmts{GormModel: (*ComplianceOperatorScanSettingBindings)(nil), Children: []*postgres.CreateStmts{}}
	CreateTableComplianceOperatorScansStmt               = &postgres.CreateStmts{GormModel: (*ComplianceOperatorScans)(nil), Children: []*postgres.CreateStmts{}}
)

type ComplianceOperatorCheckResults struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}

type ComplianceOperatorProfiles struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}

type ComplianceOperatorRules struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}

type ComplianceOperatorScanSettingBindings struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}

type ComplianceOperatorScans struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
