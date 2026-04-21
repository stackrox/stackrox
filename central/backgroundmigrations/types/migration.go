package types

import (
	"context"

	"github.com/stackrox/rox/pkg/postgres"
)

// BackgroundMigration defines a long-running migration that runs in Central after startup.
// Examples for long running migration use cases: Backfilling a collumn from serialized values, Index Creation
type BackgroundMigration struct {
	// StartingSeqNum is the seqnum before this migration runs.
	StartingSeqNum int
	// VersionAfterSeqNum is StartingSeqNum + 1.
	VersionAfterSeqNum int
	// Description is a human-readable description of the migration.
	Description string
	// Run executes the migration. Contract:
	// - Must be idempotent (safe to re-run after rollback)
	// - Must check ctx.Done() between units of work for graceful shutdown
	// - Must be consistent after re-run, previously migrated rows need to be rechecked on a re-run
	Run func(ctx context.Context, db postgres.DB) error
}
