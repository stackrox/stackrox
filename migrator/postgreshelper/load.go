package postgreshelper

import (
	"fmt"
	"os"
	"time"

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
func Load(conf *config.Config) (*gorm.DB, error) {
	password, err := os.ReadFile(dbPasswordFile)
	if err != nil {
		log.WriteToStderrf("pgsql: could not load password file %q: %v", dbPasswordFile, err)
		return nil, err
	}
	source := fmt.Sprintf("%s password=%s", conf.CentralDB.Source, password)
	source = pgutils.PgxpoolDsnToPgxDsn(source)

	// Central waits for central-db ready with retries
	var gormDB *gorm.DB
	err = retry.WithRetry(func() error {
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
	}
	return gormDB, err
}
