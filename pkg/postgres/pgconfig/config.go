package pgconfig

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	// DBPasswordFile is the database password file
	DBPasswordFile = "/run/secrets/stackrox.io/db-password/password"

	activeSuffix = "_active"

	// capacity - Minimum recommended Postgres capacity
	capacity = 100 * size.GB

	connectTimeout = 15 * time.Second
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
	config.ConnConfig.ConnectTimeout = connectTimeout

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
