package metadata

import (
	"github.com/stackrox/rox/pkg/migrations"
)

const (
	// ErrUnableToRestore -- cannot restore upgraded backup to a downgraded central
	ErrUnableToRestore = "The backup bundle being restored is from an upgraded version of central and thus cannot applied.  The restored version %s, current version %s"

	// ErrSoftwareNotCompatibleWithDatabase -- downgrade is not supported as software is incompatible with the data.
	ErrSoftwareNotCompatibleWithDatabase = "Software downgrade is not supported.  The software supports database migration version of %d but the database requires the software support a database migration version to be at least %d which is software version %s"
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

// NewPostgres returns a new ready-to-use store.
func NewPostgres(migVer *migrations.MigrationVersion, databaseName string) *DBClone {
	return &DBClone{migVer: migVer, databaseName: databaseName}
}
