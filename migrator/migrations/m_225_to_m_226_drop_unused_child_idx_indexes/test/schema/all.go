// Package schema contains frozen pre-migration GORM models for all tables
// that had standalone child _idx indexes. These models reproduce the old index
// tags so GORM AutoMigrate creates the indexes that the migration drops.

package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

// AllCreateStmts aggregates every table's CreateStmts for use in test setup.
var AllCreateStmts = []*postgres.CreateStmts{
	CreateTableAuthMachineToMachineConfigsStmt,
	CreateTableBaseImagesStmt,
	CreateTableCollectionsStmt,
	CreateTableComplianceOperatorProfileV2Stmt,
	CreateTableComplianceOperatorReportSnapshotV2Stmt,
	CreateTableComplianceOperatorRuleV2Stmt,
	CreateTableComplianceOperatorScanConfigurationV2Stmt,
	CreateTableDeploymentsStmt,
	CreateTableImagesStmt,
	CreateTableImagesV2Stmt,
	CreateTableNodesStmt,
	CreateTablePodsStmt,
	CreateTableReportConfigurationsStmt,
	CreateTableRoleBindingsStmt,
	CreateTableSecretsStmt,
	CreateTableVulnerabilityRequestsStmt,
}
