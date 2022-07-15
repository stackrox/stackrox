package pgconfig

import (
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	dbPasswordFile = "/run/secrets/stackrox.io/db-password/password"

	activeSuffix = "_active"
)

// GetPostgresConfig - gets the configuration used to connect to Postgres
func GetPostgresConfig() (map[string]string, *pgxpool.Config, error) {
	centralConfig := config.GetConfig()
	password, err := os.ReadFile(dbPasswordFile)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "pgsql: could not load password file %q", dbPasswordFile)
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
	return fmt.Sprintf("%s%s", config.GetConfig().CentralDB.RootDatabaseName, activeSuffix)
}
