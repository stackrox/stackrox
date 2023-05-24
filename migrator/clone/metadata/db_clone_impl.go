package metadata

import (
	"github.com/stackrox/rox/pkg/migrations"
)

const (
	// ErrNoPrevious - cannot rollback
	ErrNoPrevious = "Downgrade is not supported. No previous database for force rollback."

	// ErrNoPreviousInDevEnv -- Downgrade is not supported in dev
	ErrNoPreviousInDevEnv = `
Downgrade is not supported.
We compare dev builds by their release tags. For example, 3.0.58.x-58-g848e7365da is greater than
3.0.58.x-57-g848e7365da. However if the dev builds are on diverged branches, the sequence could be wrong.
These builds are not comparable.

To address this:
1. if you are testing migration, you can merge or rebase to make sure the builds are not diverged; or
2. if you simply want to switch the image, you can disable upgrade rollback and bypass this check by:
kubectl -n stackrox set env deploy/central ROX_DONT_COMPARE_DEV_BUILDS=true
`

	// ErrForceUpgradeDisabled -- force rollback is disabled
	ErrForceUpgradeDisabled = "Central force rollback is disabled. If you want to force rollback to the database before last upgrade, please enable force rollback to current version in central config. Note: all data updates since last upgrade will be lost."

	// ErrPreviousMismatchWithVersions -- downgrade is not supported as previous version is too many versions back.
	ErrPreviousMismatchWithVersions = "Database downgrade is not supported. We can only rollback to the central version before last upgrade. Last upgrade %s, current version %s"

	// ErrUnableToRestore -- cannot restore upgraded backup to a downgraded central
	ErrUnableToRestore = "The backup bundle being restored is from an upgraded version of central and thus cannot applied.  The restored version %s, current version %s"

	// ErrSoftwareNotCompatibleWithDatabase -- downgrade is not supported as software is incompatible with the data.
	ErrSoftwareNotCompatibleWithDatabase = "Software downgrade is not supported.  The software supports database version of %d but the database requires the software support a database version to be at least least %d"
)

// DBClone -- holds information related to DB clones
type DBClone struct {
	dirName      string
	migVer       *migrations.MigrationVersion
	databaseName string
}

// GetVersion -- returns the version associated with the clone.
func (d *DBClone) GetVersion() string {
	if d.migVer == nil {
		return ""
	}
	return d.migVer.MainVersion
}

// GetSeqNum -- returns the sequence number associated with the clone.
func (d *DBClone) GetSeqNum() int {
	if d.migVer == nil {
		return 0
	}
	return d.migVer.SeqNum
}

// GetMinimumSeqNum -- returns the minimum sequence number supported by the database.
func (d *DBClone) GetMinimumSeqNum() int {
	if d.migVer == nil {
		return 0
	}
	return d.migVer.MinimumSeqNum
}

// GetDirName -- returns the file system location of the clone.  (Only valid pre-Postgres)
func (d *DBClone) GetDirName() string {
	return d.dirName
}

// GetDatabaseName -- returns the database name of the clone.  (Postgres)
func (d *DBClone) GetDatabaseName() string {
	return d.databaseName
}

// GetMigVersion -- returns the migration version associated with the clone.
func (d *DBClone) GetMigVersion() *migrations.MigrationVersion {
	return d.migVer
}

// New returns a new ready-to-use store.
func New(dirName string, migVer *migrations.MigrationVersion) *DBClone {
	return &DBClone{dirName: dirName, migVer: migVer, databaseName: ""}
}

// NewPostgres returns a new ready-to-use store.
func NewPostgres(migVer *migrations.MigrationVersion, databaseName string) *DBClone {
	return &DBClone{migVer: migVer, databaseName: databaseName}
}
