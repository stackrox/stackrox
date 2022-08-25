package postgreshelper

import (
	"context"
	"fmt"
	"strings"
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
func Load(conf *config.Config, databaseName string) (*pgxpool.Pool, *gorm.DB, error) {
	log.WriteToStderrf("Load database = %q", databaseName)
	once.Do(func() {
		ctx := context.Background()

		sourceMap, adminConfig, err := pgconfig.GetPostgresConfig()
		if err != nil {
			return
		}
		// Create the central database if necessary
		if !pgadmin.CheckIfDBExists(adminConfig, databaseName) {
			err = pgadmin.CreateDB(sourceMap, adminConfig, pgadmin.AdminDB, databaseName)
			if err != nil {
				log.WriteToStderrf("Could not create central database: %v", err)
				return
			}
		}

		// Add the active database and password to the source
		gormSource := fmt.Sprintf("%s password=%s database=%s client_encoding=UTF-8", conf.CentralDB.Source, adminConfig.ConnConfig.Password, databaseName)
		gormSource = pgutils.PgxpoolDsnToPgxDsn(gormSource)

		// Waits for central-db ready with retries
		err = retry.WithRetry(func() error {
			if postgresDB == nil {
				// Clone config to connect to template DB
				tempConfig := adminConfig.Copy()

				// Need to connect on a static DB so we can rename the used DBs.
				tempConfig.ConnConfig.Database = databaseName

				postgresDB, err = pgxpool.ConnectConfig(ctx, tempConfig)
				if err != nil {
					log.WriteToStderrf("fail to connect to central db %v", err)
					return err
				}
			}
			log.WriteToStderrf("connect to gorm: %v", strings.Replace(gormSource, adminConfig.ConnConfig.Password, "<REDACTED>", -1))

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
			sqlDB, err := gormDB.DB()
			if err != nil {
				return
			}
			if sqlDB != nil {
				utils.IgnoreError(sqlDB.Close)
			}
		}
		if postgresDB != nil {
			postgresDB.Close()
		}
	})
}
