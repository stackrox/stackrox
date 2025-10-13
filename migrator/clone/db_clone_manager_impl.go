package clone

import (
	"github.com/pkg/errors"
	pgClone "github.com/stackrox/rox/migrator/clone/postgres"
	"github.com/stackrox/rox/migrator/clone/rocksdb"
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
func (d *dbCloneManagerImpl) GetCloneToMigrate() (string, error) {
	var pgClone string
	var migrateFromRocks bool
	var err error

	// We have to support the restoration of legacy backups for a couple of releases.  This allows us to determine
	// if we are dealing with that case.
	if d.dbmRocks.CheckForRestore() {
		return "", errors.New("Effective release 4.5, restores from pre-4.0 releases are no longer supported.")
	}

	pgClone, migrateFromRocks, err = d.dbmPostgres.GetCloneToMigrate(d.dbmRocks.GetVersion(rocksdb.CurrentClone))
	if err != nil {
		return "", err
	}

	// If we are doing an upgrade from 3.74 or prior to 4.5 or later we throw an error as that is no longer supported.
	if migrateFromRocks {
		return "", errors.New("Effective release 4.5, upgrades from pre-4.0 releases are no longer supported.")
	}

	return pgClone, nil
}

// Persist - replaces current clone with upgraded one.
func (d *dbCloneManagerImpl) Persist(pgClone string) error {
	// External DB does not use clone copies so simply return once migration is complete
	if d.external {
		return nil
	}

	return d.dbmPostgres.Persist(pgClone)
}
