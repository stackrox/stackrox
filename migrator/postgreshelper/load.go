package postgreshelper

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	postgresConnectionRetries    = 18
	postgresConnectRetryInterval = 10 * time.Second
)

var (
	postgresDB *pgxpool.Pool
	gormDB     *gorm.DB

	err error

	once      sync.Once
	closeOnce sync.Once
)

// Load loads a Postgres instance and returns a GormDB.
func Load(conf *config.Config) (*pgxpool.Pool, *gorm.DB, error) {
	once.Do(func() {
		var password []byte
		ctx := context.Background()

		activeDB := pgconfig.GetActiveDB()

		sourceMap, adminConfig, err := pgconfig.GetPostgresConfig()
		if err != nil {
			return
		}
		// Create the central database if necessary
		if !pgadmin.CheckIfDBExists(adminConfig, activeDB) {
			err = pgadmin.CreateDB(sourceMap, adminConfig, pgadmin.AdminDB, activeDB)
			if err != nil {
				log.WriteToStderrf("Could not create central database: %v", err)
				return
			}
		}
		adminConfig.ConnConfig.Database = activeDB

		// Add the active database and password to the source
		gormSource := fmt.Sprintf("%s password=%s database=%s", conf.CentralDB.Source, password, activeDB)
		gormSource = pgutils.PgxpoolDsnToPgxDsn(gormSource)

		// Waits for central-db ready with retries
		err = retry.WithRetry(func() error {
			var err error
			if postgresDB == nil {
				postgresDB, err = pgxpool.ConnectConfig(ctx, adminConfig)
				if err != nil {
					log.WriteToStderrf("fail to connect to central db %v", err)
					return err
				}
			}
			gormDB, err = gorm.Open(postgres.Open(gormSource), &gorm.Config{
				NamingStrategy:  pgutils.NamingStrategy,
				CreateBatchSize: 1000})
			return err
		}, retry.Tries(postgresConnectionRetries), retry.BetweenAttempts(func(attempt int) {
			time.Sleep(postgresConnectRetryInterval)
		}), retry.OnFailedAttempts(func(err error) {
			log.WriteToStderrf("failed to connect to central database: %v", err)
		}))

		if err != nil {
			log.WriteToStderrf("timed out connecting to database: %v", err)
		} else {
			log.WriteToStderr("Successfully connected to central database.")
		}
	})

	return postgresDB, gormDB, err
}

// Close closes postgres databases
func Close() {
	closeOnce.Do(func() {
		if gormDB != nil {
			sqlDB, _ := gormDB.DB()
			if sqlDB != nil {
				utils.IgnoreError(sqlDB.Close)
			}
		}
		if postgresDB != nil {
			postgresDB.Close()
		}
	})
}
