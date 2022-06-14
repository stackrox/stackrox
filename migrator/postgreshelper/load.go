package postgreshelper

import (
	"fmt"
	"os"
	"time"

	"github.com/stackrox/stackrox/migrator/log"
	"github.com/stackrox/stackrox/pkg/config"
	"github.com/stackrox/stackrox/pkg/postgres/pgutils"
	"github.com/stackrox/stackrox/pkg/retry"
	"github.com/stackrox/stackrox/pkg/sync"
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

var (
	gormDB *gorm.DB
	once   sync.Once
	err    error
)

// Load loads a Postgres instance and returns a GormDB.
func Load(conf *config.Config) (*gorm.DB, error) {
	once.Do(func() {
		var password []byte
		password, err = os.ReadFile(dbPasswordFile)
		if err != nil {
			log.WriteToStderrf("pgsql: could not load password file %q: %v", dbPasswordFile, err)
			return
		}
		source := fmt.Sprintf("%s password=%s", conf.CentralDB.Source, password)
		source = pgutils.PgxpoolDsnToPgxDsn(source)

		// Waits for central-db ready with retries
		err = retry.WithRetry(func() error {
			gormDB, err = gorm.Open(postgres.Open(source), &gorm.Config{
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
		}
	})
	return gormDB, err
}
