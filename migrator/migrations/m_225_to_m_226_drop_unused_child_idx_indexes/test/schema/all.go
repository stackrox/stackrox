package schema

import "github.com/stackrox/rox/pkg/postgres"

// AllCreateStmts contains the frozen CreateStmts for all tables
// that had standalone _idx indexes on child table idx columns.
// Dependency tables (Roles, Notifiers, BaseImageRepositories) are listed first
// because other tables have FK references to them.
var AllCreateStmts = []*postgres.CreateStmts{
	CreateTableRolesStmt,
	CreateTableNotifiersStmt,
	CreateTableBaseImageRepositoriesStmt,
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
