package postgreshelper

import (
	"fmt"
	"os"

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

// Load loads a Postgres instance and returns a GormDB.
func Load(conf *config.Config) (*gorm.DB, error) {
	password, err := os.ReadFile(dbPasswordFile)
	if err != nil {
		log.WriteToStderrf("pgsql: could not load password file %q: %v", dbPasswordFile, err)
		return nil, err
	}
	source := fmt.Sprintf("%s password=%s", conf.CentralDB.Source, password)
	source = pgutils.PgxpoolDsnToPgxDsn(source)
	log.WriteToStderrf(source)

	gormDB, err := gorm.Open(postgres.Open(source), &gorm.Config{
		NamingStrategy:  pgutils.NamingStrategy,
		CreateBatchSize: 1000})
	if err != nil {
		return nil, errors.Wrap(err, "failed to open postgres db")
	}
	return gormDB, nil
}
