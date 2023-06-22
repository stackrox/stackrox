package types

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
)

var (
	// DefaultMigrationTimeout -- default timeout for migration postgres statements
	DefaultMigrationTimeout = env.PostgresDefaultMigrationStatementTimeout.DurationSetting()
)

// A Migration represents a migration.
type Migration struct {
	// StartingSeqNum is the required seq num before the migration runs.
	StartingSeqNum int
	// Run runs the migration, given the instance of the DB, returning an error if it doesn't work.
	// Run is NOT responsible for validating that the DB is of the right version,
	// It can safely assume that, if it has been called, the DB is of the version it expects
	// It is also NOT responsible for writing the updated version to the DB on conclusion -- that logic
	// exists in the runner, and does not need to be included in every migration.
	Run func(databases *Databases) error
	// The VersionAfter is the version put into the DB after the migration runs.
	// The seq num in VersionAfter MUST be one greater than the StartingSeqNum of this migration.
	// All other (optional) metadata can be whatever the user desires, and has no bearing on the
	// functioning of the migrator.
	VersionAfter *storage.Version
}
