package postgres

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// CurrentClone - active postgres clone name
	CurrentClone = migrations.CurrentDatabase

	// RestoreClone - restore postgres clone name
	RestoreClone = migrations.RestoreDatabase

	// BackupClone - backup postgres clone name
	BackupClone = migrations.BackupDatabase

	// PreviousClone - previous postgres clone used for rollback
	PreviousClone = migrations.PreviousDatabase

	// TempClone - temp postgres clone
	TempClone = "central_temp"
)

var (
	knownClones = set.NewStringSet(CurrentClone, RestoreClone, BackupClone, PreviousClone)

	log = logging.CurrentModule().Logger()
)

// DBCloneManager - scans and manage database clone within central.
type DBCloneManager interface {
	// Scan - Looks for database Clones
	Scan() error

	// GetCloneToMigrate -- retrieves the clone that needs moved to the active database.
	GetCloneToMigrate(rocksVersion *migrations.MigrationVersion, restoreFromRocks bool) (string, bool, error)

	// Persist -- moves the clone database to be the active database.
	Persist(clone string) error

	// GetCurrentVersion -- gets the version of the current clone
	GetCurrentVersion() *migrations.MigrationVersion
}
