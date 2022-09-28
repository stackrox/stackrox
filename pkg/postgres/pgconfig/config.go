package pgconfig

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	// DBPasswordFile is the database password file
	DBPasswordFile = "/run/secrets/stackrox.io/db-password/password"

	activeSuffix = "_active"

	// capacity - Minimum recommended Postgres capacity
	capacity = 100 * size.GB

	// AdminDB - name of admin database
	AdminDB = "postgres"

	// EmptyDB - name of an empty database (automatically created by postgres)
	EmptyDB = "template0"

	// postgresOpenRetries - number of retries when trying to open a connection
	postgresOpenRetries = 10

	// postgresTimeBetweenRetries - time to wait between retries
	postgresTimeBetweenRetries = 10 * time.Second
)

// GetPostgresConfig - gets the configuration used to connect to Postgres
func GetPostgresConfig() (map[string]string, *pgxpool.Config, error) {
	centralConfig := config.GetConfig()
	password, err := os.ReadFile(DBPasswordFile)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "pgsql: could not load password file %q", DBPasswordFile)
	}
	// Add the password to the source to pass to get the pool config
	source := fmt.Sprintf("%s password=%s", centralConfig.CentralDB.Source, password)

	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not parse postgres config")
	}

	sourceMap, err := ParseSource(source)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not parse postgres source")
	}

	return sourceMap, config, nil
}

// ParseSource - parses the source string into a map for simpler access by commands
func ParseSource(source string) (map[string]string, error) {
	if source == "" {
		return nil, errors.New("source string is empty")
	}

	sourceSlice := strings.Fields(source)
	sourceMap := make(map[string]string)
	for _, pair := range sourceSlice {
		// Due to the possibility that the password could potentially have an = we
		// need to ensure that we get the entire password
		key, value := stringutils.Split2(pair, "=")

		sourceMap[key] = strings.TrimSpace(value)
	}

	return sourceMap, nil
}

// GetActiveDB - returns the name of the active database
func GetActiveDB() string {
	return fmt.Sprintf("%s%s", config.GetConfig().CentralDB.DatabaseName, activeSuffix)
}

// GetPostgresCapacity - returns the capacity of the Postgres instance
func GetPostgresCapacity() int64 {
	return capacity
}

// GetAdminPool - returns a pool to connect to the admin database.
// This is useful for renaming databases such as a restore to active.
// THIS POOL SHOULD BE CLOSED ONCE ITS PURPOSE HAS BEEN FULFILLED.
func GetAdminPool(postgresConfig *pgxpool.Config) *pgxpool.Pool {
	// Clone config to connect to template DB
	tempConfig := postgresConfig.Copy()

	// Need to connect on a static DB so we can rename the used DBs.
	tempConfig.ConnConfig.Database = AdminDB

	postgresDB := getPool(tempConfig)

	return postgresDB
}

// GetClonePool - returns a connection pool for the specified database clone.
// THIS POOL SHOULD BE CLOSED ONCE ITS PURPOSE HAS BEEN FULFILLED.
func GetClonePool(postgresConfig *pgxpool.Config, clone string) *pgxpool.Pool {
	// Clone config to connect to template DB
	tempConfig := postgresConfig.Copy()

	// Need to connect on a static DB so we can rename the used DBs.
	tempConfig.ConnConfig.Database = clone

	postgresDB := getPool(tempConfig)

	return postgresDB
}

func getPool(postgresConfig *pgxpool.Config) *pgxpool.Pool {
	var err error
	var postgresDB *pgxpool.Pool

	if err := retry.WithRetry(func() error {
		postgresDB, err = pgxpool.ConnectConfig(context.Background(), postgresConfig)
		return err
	}, retry.Tries(postgresOpenRetries), retry.BetweenAttempts(func(attempt int) {
		time.Sleep(postgresTimeBetweenRetries)
	}), retry.OnFailedAttempts(func(err error) {
	})); err != nil {
	}

	return postgresDB
}
