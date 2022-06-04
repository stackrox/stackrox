package postgreshelper

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
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
	postgresDB, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		log.WriteToStderrf("fail to connect to central db %v", err)
		return nil, nil, err
	}
	source = pgutils.PgxpoolDsnToPgxDsn(source)
	log.WriteToStderrf(source)

	gormDB, err := gorm.Open(postgres.Open(source), &gorm.Config{
		NamingStrategy:  pgutils.NamingStrategy,
		CreateBatchSize: 1000})
	if err != nil {
		postgresDB.Close()
		return nil, nil, errors.Wrap(err, "failed to open postgres db")
	}
	return gormDB.WithContext(ctx), postgresDB, nil
}
