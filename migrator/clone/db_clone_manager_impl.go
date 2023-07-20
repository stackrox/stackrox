package clone

import (
	pgClone "github.com/stackrox/rox/migrator/clone/postgres"
	"github.com/stackrox/rox/migrator/clone/rocksdb"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
)

// dbCloneManagerImpl - scans and manage database clones within central.
type dbCloneManagerImpl struct {
	forceRollbackVersion string
	adminConfig          *postgres.Config
	sourceMap            map[string]string
	basePath             string
	dbmRocks             rocksdb.DBCloneManager
	dbmPostgres          pgClone.DBCloneManager
	external             bool
}

// NewPostgres - returns a new ready-to-use manager.
func NewPostgres(basePath string, forceVersion string, adminConfig *postgres.Config, sourceMap map[string]string) DBCloneManager {
	return &dbCloneManagerImpl{
		forceRollbackVersion: forceVersion,
		adminConfig:          adminConfig,
		sourceMap:            sourceMap,
		basePath:             basePath,
		dbmRocks:             rocksdb.New(basePath, forceVersion),
		dbmPostgres:          pgClone.New(forceVersion, adminConfig, sourceMap),
		external:             pgconfig.IsExternalDatabase(),
	}
}

// Scan - checks the persistent data of central and gather the clone information
// from the supported databases.
func (d *dbCloneManagerImpl) Scan() error {
	err := d.dbmRocks.Scan()
	if err != nil {
		// If our focus is Postgres, just log the error and ignore Rocks
		log.Warn(err)
	}

	return d.dbmPostgres.Scan()
}

// GetCloneToMigrate - finds a clone to migrate.
// It returns the clone link, path to database, postgres database name and error if fails.
func (d *dbCloneManagerImpl) GetCloneToMigrate() (string, string, string, error) {
	var pgClone string
	var migrateFromRocks bool
	var err error

	// We have to support the restoration of legacy backups for a couple of releases.  This allows us to determine
	// if we are dealing with that case.
	restoreFromRocks := d.dbmRocks.CheckForRestore()

	// Get the version of the Rocks Current so Postgres manager can use that info
	// to determine what clone it needs to migrate.
	var rocksVersion *migrations.MigrationVersion
	if restoreFromRocks {
		rocksVersion = d.dbmRocks.GetVersion(rocksdb.RestoreClone)
	} else {
		rocksVersion = d.dbmRocks.GetVersion(rocksdb.CurrentClone)
	}

	pgClone, migrateFromRocks, err = d.dbmPostgres.GetCloneToMigrate(rocksVersion, restoreFromRocks)
	if err != nil {
		return "", "", "", err
	}

	// If we need to migrate from rocks we need to continue processing and
	// get the Rocks clones.  If we don't, there is no need to process Rocks, but
	// we will check to see if we can get rid of rocks
	if !migrateFromRocks {
		return "", "", pgClone, nil
	}

	// Get the RocksDB clone we are migrating
	clone, clonePath, err := d.dbmRocks.GetCloneToMigrate()
	if err != nil {
		if migrateFromRocks {
			return "", "", "", err
		}
		log.Warnf("unable to determine Rocks clone.  Continuing with postgres.  %v", err)
	}

	return clone, clonePath, pgClone, nil
}

// Persist - replaces current clone with upgraded one.
func (d *dbCloneManagerImpl) Persist(cloneName string, pgClone string, persistBoth bool) error {
	// We need to persist the Rocks previous, so it is there in case of a rollback.  In the case of
	// an upgrade that will generate a previous, the Temp Clone will be the one RocksDB persists.
	// During the persist operation the Current clone will move to Previous and Temp will move to Current.
	if persistBoth && (cloneName == rocksdb.TempClone || cloneName == rocksdb.RestoreClone) {
		if err := d.dbmRocks.Persist(cloneName); err != nil {
			log.Warnf("Unable to create a previous version of Rocks to rollback to: %v", err)
		}
	}

	// External DB does not use clone copies so simply return once migration is complete
	if d.external {
		return nil
	}

	return d.dbmPostgres.Persist(pgClone)
}
