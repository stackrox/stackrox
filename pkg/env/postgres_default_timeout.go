package env

import "time"

var (
	// PostgresDefaultStatementTimeout sets the default timeout for Postgres statements
	PostgresDefaultStatementTimeout = registerDurationSetting("ROX_POSTGRES_DEFAULT_TIMEOUT", 60*time.Second)

	// PostgresDefaultCursorTimeout sets the default timeout for Postgres cursor statements
	PostgresDefaultCursorTimeout = registerDurationSetting("ROX_POSTGRES_DEFAULT_CURSOR_TIMEOUT", 10*time.Minute)

	// PostgresDefaultMigrationStatementTimeout sets the default timeout for Postgres statements during migration
	PostgresDefaultMigrationStatementTimeout = registerDurationSetting("ROX_POSTGRES_MIGRATION_STATEMENT_TIMEOUT", 20*time.Minute)
)
