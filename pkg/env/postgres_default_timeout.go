package env

import "time"

var (
	// PostgresDefaultStatementTimeout sets the default timeout for Postgres statements. This does not
	// set the statement_timeout, usually configured in the central-external-db configmap. Statement
	// will fail after either of each is exceeded.
	PostgresDefaultStatementTimeout = registerDurationSetting("ROX_POSTGRES_DEFAULT_TIMEOUT", 60*time.Second)

	// PostgresDefaultCursorTimeout sets the default timeout for Postgres cursor statements
	PostgresDefaultCursorTimeout = registerDurationSetting("ROX_POSTGRES_DEFAULT_CURSOR_TIMEOUT", 10*time.Minute)

	// PostgresDefaultMigrationStatementTimeout sets the default timeout for Postgres statements during migration
	PostgresDefaultMigrationStatementTimeout = registerDurationSetting("ROX_POSTGRES_MIGRATION_STATEMENT_TIMEOUT", 2*time.Hour)

	// PostgresDefaultNetworkFlowDeleteTimeout sets the default timeout for deleting network flows
	PostgresDefaultNetworkFlowDeleteTimeout = registerDurationSetting("ROX_POSTGRES_NETWORK_FLOW_DELETE_TIMEOUT", 3*time.Minute)

	// PostgresDefaultNetworkFlowQueryTimeout sets the default timeout for querying network flows
	PostgresDefaultNetworkFlowQueryTimeout = registerDurationSetting("ROX_POSTGRES_NETWORK_FLOW_QUERY_TIMEOUT", 3*time.Minute)

	// PostgresDefaultPruningStatementTimeout sets the default timeout for pruning operations
	PostgresDefaultPruningStatementTimeout = registerDurationSetting("ROX_POSTGRES_DEFAULT_PRUNING_TIMEOUT", 3*time.Minute)

	// How often to perform pruning
	PruneInterval = registerDurationSetting("ROX_PRUNE_INTERVAL", 1*time.Hour)

	// Timeout for reading part of pruning query
	PruneOrphanedQueryTimeout = registerDurationSetting("ROX_PRUNE_ORPHANED_QUERY_TIMEOUT", 5*time.Minute)

	// Pruning is necessary due to the asynchronous nature of some events. E.g.
	// a process indicator may arrive before a corresponding deployment was
	// received, which will prevent us from making any hard constraints and
	// make data cleanup more complicated. Pruner helps to address that by
	// defining an arbitrary threshold, after which an event is concidered to
	// be orphaned and could be deleted.
	//
	// Currently the default timeout is 5 minutes, which is a plausible looking
	// arbitrary value. This cut off line may significantly affect the
	// underlying query performance, if it doesn't match the actual lifetime:
	//   * if it's too short and the actual events live longer, the pruner
	//     would have to read lots of non-orphaned data only to skip it.
	//   * if it's too long, orphaned records will stay longer, clogging the
	//     table.
	//
	// Thus it has to be configured if needed, when pruner performance issues
	// are observed.
	PruneOrphanedWindow = registerDurationSetting("ROX_PRUNE_ORPHANED_WINDOW", 30*time.Minute)

	// PostgresVMStatementTimeout sets the statement timeout for VM
	PostgresVMStatementTimeout = registerDurationSetting("ROX_POSTGRES_VM_STATEMENT_TIMEOUT", 3*time.Minute)
)
