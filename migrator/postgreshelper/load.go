package postgreshelper

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackrox/rox/migrator/log"
	migGorm "github.com/stackrox/rox/migrator/postgres/gorm"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/sync"
	"gorm.io/gorm"
)

var (
	postgresDB *pgxpool.Pool
	gormDB     *gorm.DB

	err error

	once      sync.Once
	closeOnce sync.Once
)

// Load loads a Postgres instance and returns a GormDB.
func Load(databaseName string) (*pgxpool.Pool, *gorm.DB, error) {
	log.WriteToStderrf("Load database = %q", databaseName)
	gc := migGorm.GetConfig()

	sourceMap, adminConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return nil, nil, err
	}
	// Create the central database if necessary
	if !pgadmin.CheckIfDBExists(adminConfig, databaseName) {
		err = pgadmin.CreateDB(sourceMap, adminConfig, pgadmin.EmptyDB, databaseName)
		if err != nil {
			log.WriteToStderrf("Could not create central database: %v", err)
			return nil, nil, err
		}
	}
	// Waits for central-db ready with retries
	postgresDB = pgadmin.GetClonePool(adminConfig, databaseName)
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
