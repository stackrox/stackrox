package clone

import (
	pgClone "github.com/stackrox/rox/migrator/clone/postgres"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
)

// dbCloneManagerImpl - scans and manage database clones within central.
type dbCloneManagerImpl struct {
	forceRollbackVersion string
	adminConfig          *postgres.Config
	sourceMap            map[string]string
	dbmPostgres          pgClone.DBCloneManager
	external             bool
}

// NewPostgres - returns a new ready-to-use manager.
func NewPostgres(forceVersion string, adminConfig *postgres.Config, sourceMap map[string]string) DBCloneManager {
	return &dbCloneManagerImpl{
		forceRollbackVersion: forceVersion,
		adminConfig:          adminConfig,
		sourceMap:            sourceMap,
		dbmPostgres:          pgClone.New(forceVersion, adminConfig, sourceMap),
		external:             pgconfig.IsExternalDatabase(),
	}
}

// Scan - checks the persistent data of central and gather the clone information
// from the supported databases.
func (d *dbCloneManagerImpl) Scan() error {
	return d.dbmPostgres.Scan()
}

// GetCloneToMigrate - finds a clone to migrate.
// It returns the postgres database name and error if fails.
func (d *dbCloneManagerImpl) GetCloneToMigrate() (string, error) {
	var pgClone string
	var err error

	pgClone, err = d.dbmPostgres.GetCloneToMigrate()
	if err != nil {
		return "", err
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
