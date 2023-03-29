package rocksdb

import (
	"regexp"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// Indexes
	bleveIndex = "scorch.bleve"
	index      = "index"

	// CurrentClone - active rocksdb clone name
	CurrentClone = migrations.Current

	// RestoreClone - restore rocksdb clone name
	RestoreClone = ".restore"

	// BackupClone - backup rocksdb clone name
	BackupClone = ".backup"

	// PreviousClone - previous rocksdb clone used for rollback
	PreviousClone = ".previous"

	// TempClone - temp rocksdb clone
	TempClone = "temp"
)

var (
	upgradeRegex = regexp.MustCompile(`^\.db-*`)
	restoreRegex = regexp.MustCompile(`^\.restore-*`)
	knownClones  = set.NewStringSet(CurrentClone, RestoreClone, BackupClone, PreviousClone)

	log = logging.CurrentModule().Logger()
)

// DBCloneManager - scans and manage database clones within central.
type DBCloneManager interface {
	// Scan - Looks for database clones
	Scan() error

	// GetCloneToMigrate -- retrieves the clone that needs moved to the actived database.
	GetCloneToMigrate() (string, string, error)

	// Persist -- moves the clone database to be the active database.
	Persist(cloneName string) error

	// GetVersion -- gets the version of the clone
	GetVersion(cloneName string) *migrations.MigrationVersion

	// GetDirName - gets the directory name of the clone
	GetDirName(cloneName string) string

	// CheckForRestore -- checks to see if a restore from a RocksDB is requested
	CheckForRestore() bool
}
