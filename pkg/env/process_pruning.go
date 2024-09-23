package env

import "time"

var (
	// ProcessPruningEnabled toggles whether process pruning should be done periodically using heuristics
	// This may be useful in certain large environments, but is fairly expensive as it requires a full
	// sweep over the database
	ProcessPruningEnabled = RegisterBooleanSetting("ROX_PROCESS_PRUNING", false)

	// How often to perform pruning
	PruneInterval = registerDurationSetting("ROX_PRUNE_INTERVAL", 1*time.Hour)

	// Timeout for reading part of pruning query
	PruneOrphanedTimeout = registerDurationSetting("ROX_PRUNE_ORPHANED_TIMEOUT", 5*time.Minute)

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
)
