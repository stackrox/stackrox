package clone

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/migrator/clone/postgres"
	"github.com/stackrox/rox/migrator/clone/rocksdb"
	"github.com/stackrox/rox/pkg/features"
)

// DBCloneManagerImpl - scans and manage database clones within central.
type dbCloneManagerImpl struct {
	forceRollbackVersion string
	adminConfig          *pgxpool.Config
	sourceMap            map[string]string
	basePath             string
	dbmRocks             rocksdb.DBCloneManager
	dbmPostgres          postgres.DBCloneManager
}

// NewPostgres - returns a new ready-to-use manager.
func NewPostgres(basePath string, forceVersion string, adminConfig *pgxpool.Config, sourceMap map[string]string) DBCloneManager {
	return &dbCloneManagerImpl{
		forceRollbackVersion: forceVersion,
		adminConfig:          adminConfig,
		sourceMap:            sourceMap,
		basePath:             basePath,
		dbmRocks:             rocksdb.New(basePath, forceVersion),
		dbmPostgres:          postgres.New(forceVersion, adminConfig, sourceMap),
	}
}

// New - returns a new ready-to-use manager.
func New(basePath string, forceVersion string) DBCloneManager {
	return &dbCloneManagerImpl{
		forceRollbackVersion: forceVersion,
		basePath:             basePath,
		dbmRocks:             rocksdb.New(basePath, forceVersion),
	}
}

// Scan - checks the persistent data of central and gather the clone information
// from the supported databases.
func (d *dbCloneManagerImpl) Scan() error {
	err := d.dbmRocks.Scan()
	if err != nil {
		// If our focus is Postgres, just log the error and ignore Rocks
		if features.PostgresDatastore.Enabled() {
			log.Warn(err)
		} else {
			return err
		}
	}

	if features.PostgresDatastore.Enabled() {
		err = d.dbmPostgres.Scan()
		if err != nil {
			return err
		}
	}

	return nil
}

// GetCloneToMigrate - finds a clone to migrate.
// It returns the clone link, path to database, postgres database name and error if fails.
func (d *dbCloneManagerImpl) GetCloneToMigrate() (string, string, string, error) {
	var pgClone string
	var migrateFromRocks bool
	var err error

	if features.PostgresDatastore.Enabled() {
		// Get the version of the Rocks Current so Postgres manager can use that info
		// to determine what clone it needs to migrate.
		rocksVersion := d.dbmRocks.GetVersion(rocksdb.CurrentClone)
		if rocksVersion != nil {
			rocksVersion.LastPersisted = d.dbmRocks.GetCurrentCloneCreationTime()
		}

		pgClone, migrateFromRocks, err = d.dbmPostgres.GetCloneToMigrate(rocksVersion)
		if err != nil {
			return "", "", "", err
		}

		// If we need to migrate from rocks we need to continue processing and
		// get the Rocks clones.  If we don't, there is no need to process Rocks
		if !migrateFromRocks {
			return "", "", pgClone, nil
		}
	}

	// Get the RocksDB clone we are migrating
	clone, clonePath, err := d.dbmRocks.GetCloneToMigrate()
	if err != nil {
		if !features.PostgresDatastore.Enabled() || migrateFromRocks {
			return "", "", "", err
		}
		log.Warnf("unable to determine Rocks clone.  Continuing with postgres.  %v", err)
	}

	return clone, clonePath, pgClone, nil
}

// Persist - replaces current clone with upgraded one.
func (d *dbCloneManagerImpl) Persist(cloneName string, pgClone string, persistBoth bool) error {
	if features.PostgresDatastore.Enabled() {
		// We need to persist the Rocks previous, so it is there in case of a rollback.  In the case of
		// an upgrade that will generate a previous, the Temp Clone will be the one RocksDB persists.
		// During the persist operation the Current clone will move to Previous and Temp will move to Current.
		if persistBoth && cloneName == rocksdb.TempClone {
			if err := d.dbmRocks.Persist(cloneName); err != nil {
				log.Warnf("Unable to create a previous version of Rocks to rollback to: %v", err)
			}
		}
		return d.dbmPostgres.Persist(pgClone)
	}
	return d.dbmRocks.Persist(cloneName)
}
