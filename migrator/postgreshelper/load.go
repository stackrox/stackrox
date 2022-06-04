package postgreshelper

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/retry"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	dbPasswordFile = "/run/secrets/stackrox.io/db-password/password"
)

const (
	postgresConnectionRetries    = 18
	postgresConnectRetryInterval = 10 * time.Second
)

// Load loads a Postgres instance and returns a GormDB.
func Load(conf *config.Config) (*gorm.DB, *pgxpool.Pool, error) {
	password, err := os.ReadFile(dbPasswordFile)
	if err != nil {
		log.WriteToStderrf("pgsql: could not load password file %q: %v", dbPasswordFile, err)
		return nil, nil, err
	}
	source := fmt.Sprintf("%s password=%s", conf.CentralDB.Source, password)

	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		log.WriteToStderrf("could not parse postgres config: %v", err)
	}
	ctx := context.Background()

	source = pgutils.PgxpoolDsnToPgxDsn(source)

	// Central waits for central-db ready with retries
	var postgresDB *pgxpool.Pool
	var gormDB *gorm.DB
	err = retry.WithRetry(func() error {
		if postgresDB == nil {
			postgresDB, err = pgxpool.ConnectConfig(ctx, config)
			if err != nil {
				log.WriteToStderrf("fail to connect to central db %v", err)
				return err
			}
		}

		gormDB, err = gorm.Open(postgres.Open(source), &gorm.Config{
			NamingStrategy:  pgutils.NamingStrategy,
			CreateBatchSize: 1000})
		return err
	}, retry.Tries(postgresConnectionRetries), retry.BetweenAttempts(func(attempt int) {
		time.Sleep(postgresConnectRetryInterval)
	}), retry.OnFailedAttempts(func(err error) {
		log.WriteToStderrf("fail to connect to central database: %v", err)
	}))

	if err != nil {
		log.WriteToStderrf("timed out connecting to database: %v, is central-db alive?", err)
		if postgresDB != nil {
			postgresDB.Close()
		}
		return nil, nil, err
	}

	return gormDB.WithContext(ctx), postgresDB, nil
}
