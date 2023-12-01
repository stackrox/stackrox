package postgreshelper

import (
	"github.com/stackrox/rox/migrator/log"
	migGorm "github.com/stackrox/rox/migrator/postgres/gorm"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/sync"
	"gorm.io/gorm"
)

var (
	postgresDB postgres.DB
	gormDB     *gorm.DB

	closeOnce sync.Once
)

// GetConnections loads the configured database within the Postgres instance and returns a GormDB.
func GetConnections() (postgres.DB, *gorm.DB, error) {
	log.WriteToStderr("Load database")
	gc := migGorm.GetConfig()

	_, dbConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return nil, nil, err
	}

	if !pgconfig.IsExternalDatabase() {
		// For migrations we may have long running jobs.  Here we explicitly turn
		// off the statement timeout for the connection and will rely on the context
		// timeouts to control this.
		dbConfig.ConnConfig.RuntimeParams["statement_timeout"] = "0"
	}

	postgresDB, err = pgadmin.GetPool(dbConfig)
	if err != nil {
		log.WriteToStderrf("timed out connecting to database: %v", err)
		return nil, nil, err
	}
	gormDB, err = gc.ConnectDatabaseWithRetries()
	if err != nil {
		postgresDB.Close()
		log.WriteToStderrf("timed out connecting to database: %v", err)
		return nil, nil, err
	}

	log.WriteToStderr("Successfully connected to central database.")
	return postgresDB, gormDB, err
}

// Load loads a Postgres instance and returns a GormDB.
// TODO(ROX-18005) Deprecate this
func Load(databaseName string) (postgres.DB, *gorm.DB, error) {
	log.WriteToStderrf("Load database = %q", databaseName)
	gc := migGorm.GetConfig()

	sourceMap, adminConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return nil, nil, err
	}

	if !pgconfig.IsExternalDatabase() {
		// Create the central database if necessary
		exists, err := pgadmin.CheckIfDBExists(adminConfig, databaseName)
		if err != nil {
			log.WriteToStderrf("Could not check for central database: %v", err)
			return nil, nil, err
		}
		if !exists {
			err = pgadmin.CreateDB(sourceMap, adminConfig, pgadmin.EmptyDB, databaseName)
			if err != nil {
				log.WriteToStderrf("Could not create central database: %v", err)
				return nil, nil, err
			}
		}

		// For migrations we may have long running jobs.  Here we explicitly turn
		// off the statement timeout for the connection and will rely on the context
		// timeouts to control this.
		adminConfig.ConnConfig.RuntimeParams["statement_timeout"] = "0"
	}

	postgresDB, err = pgadmin.GetClonePool(adminConfig, databaseName)
	if err != nil {
		log.WriteToStderrf("timed out connecting to database: %v", err)
		return nil, nil, err
	}
	gormDB, err = gc.ConnectWithRetries(databaseName)
	if err != nil {
		postgresDB.Close()
		log.WriteToStderrf("timed out connecting to database: %v", err)
		return nil, nil, err
	}

	log.WriteToStderr("Successfully connected to central database.")
	return postgresDB, gormDB, err
}

// Close closes postgres databases
func Close() {
	closeOnce.Do(func() {
		migGorm.Close(gormDB)
		if postgresDB != nil {
			postgresDB.Close()
		}
	})
}
