package env

import "time"

var (
	// PostgresDefaultStatementTimeout sets the default timeout for Postgres statements
	PostgresDefaultStatementTimeout = registerDurationSetting("ROX_POSTGRES_DEFAULT_TIMEOUT", 60*time.Second)

	// PostgresDefaultCursorTimeout sets the default timeout for Postgres cursor statements
	PostgresDefaultCursorTimeout = registerDurationSetting("ROX_POSTGRES_DEFAULT_CURSOR_TIMEOUT", 10*time.Minute)

	// PostgresDefaultMigrationStatementTimeout sets the default timeout for Postgres statements during migration
	PostgresDefaultMigrationStatementTimeout = registerDurationSetting("ROX_POSTGRES_MIGRATION_STATEMENT_TIMEOUT", 2*time.Hour)

	// PostgresDefaultNetworkFlowDeleteTimeout sets the default timeout for deleting network flows
	PostgresDefaultNetworkFlowDeleteTimeout = registerDurationSetting("ROX_POSTGRES_NETWORK_FLOW_DELETE_TIMEOUT", 3*time.Minute)

	// PostgresDefaultNetworkFlowQueryTimeout sets the default timeout for querying network flows
	PostgresDefaultNetworkFlowQueryTimeout = registerDurationSetting("ROX_POSTGRES_NETWORK_FLOW_QUERY_TIMEOUT", 3*time.Minute)
)
