package pgconfig

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	// DBPasswordFile is the database password file
	DBPasswordFile = "/run/secrets/stackrox.io/db-password/password"

	// capacity - Minimum recommended Postgres capacity
	capacity = 100 * size.GB

	connectTimeout = 15 * time.Second
)

var (
	pgConfigMap  map[string]string
	pgConfig     *postgres.Config
	pgConfigErr  error
	pgConfigOnce sync.Once
)

// GetPostgresConfig - gets the configuration used to connect to Postgres
func GetPostgresConfig() (map[string]string, *postgres.Config, error) {
	pgConfigOnce.Do(func() {
		pgConfigMap, pgConfig, pgConfigErr = getPostgresConfig()
	})
	return pgConfigMap, pgConfig, pgConfigErr
}

func getPostgresConfig() (map[string]string, *postgres.Config, error) {
	centralConfig := config.GetConfig()
	password, err := os.ReadFile(DBPasswordFile)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "pgsql: could not load password file %q", DBPasswordFile)
	}
	// Add the password to the source to pass to get the pool config
	source := fmt.Sprintf("%s password=%s", centralConfig.CentralDB.Source, password)

	config, err := postgres.ParseConfig(source)
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
	return migrations.CurrentDatabase
}

// GetPostgresCapacity - returns the capacity of the Postgres instance
func GetPostgresCapacity() int64 {
	return capacity
}

// IsExternalDatabase - retrieves whether Postgres is external
func IsExternalDatabase() bool {
	return config.GetConfig().CentralDB.External
}
