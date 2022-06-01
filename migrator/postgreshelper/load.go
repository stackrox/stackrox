package postgreshelper

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	dbPasswordFile = "/run/secrets/stackrox.io/db-password/password"
)

var (
	postgresDB      *gorm.DB
	pgxPoolDSNRegex = regexp.MustCompile(`(^| )(pool_max_conns|pool_min_conns|pool_max_conn_lifetime|pool_max_conn_idle_time|pool_health_check_period)=\S+`)
)

func Load(conf *config.Config) (*gorm.DB, error) {
	password, err := os.ReadFile(dbPasswordFile)
	if err != nil {
		log.WriteToStderrf("pgsql: could not load password file %q: %v", dbPasswordFile, err)
		return nil, err
	}
	source := fmt.Sprintf("%s password=%s", conf.CentralDB.Source, password)
	source = PgxpoolDsnToPgxDsn(source)
	log.WriteToStderrf(source)

	postgresDB, err = gorm.Open(postgres.Open(source), &gorm.Config{
		NamingStrategy:  pgutils.NamingStrategy,
		CreateBatchSize: 1000})
	if err != nil {
		return nil, errors.Wrap(err, "failed to open postgres db")
	}
	return postgresDB, nil
}

func PgxpoolDsnToPgxDsn(pgxpoolDsn string) string {
	return strings.TrimSpace(pgxPoolDSNRegex.ReplaceAllString(pgxpoolDsn, ""))
}
