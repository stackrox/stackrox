package main

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/postgreshelper"
	"github.com/stackrox/rox/migrator/runner"
	"github.com/stackrox/rox/migrator/types"
	migVer "github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"gorm.io/gorm"
)

func upgrade(_ *config.Config, dbClone string, _ bool) error {
	var gormDB *gorm.DB
	var pgPool postgres.DB
	var err error
	// TODO(ROX-18005) Update to only use single DB when `central_previous` is no longer supported
	if pgconfig.IsExternalDatabase() {
		pgPool, gormDB, err = postgreshelper.GetConnections()
	} else {
		pgPool, gormDB, err = postgreshelper.Load(dbClone)
	}
	if err != nil {
		return errors.Wrap(err, "failed to connect to postgres DB")
	}
	// Close when needed
	defer postgreshelper.Close()

	ctx := sac.WithAllAccess(context.Background())
	ver, err := migVer.ReadVersionGormDB(ctx, gormDB)
	if err != nil {
		return errors.Wrap(err, "failed to get version from the database")
	}

	// If Postgres has no version and we have no bolt then we have no populated databases at all and thus don't
	// need to migrate
	if ver.SeqNum == 0 && ver.MainVersion == "0" {
		log.WriteToStderr("Fresh install of the database. There is no data to migrate...")
		pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
		return nil
	}
	log.WriteToStderrf("version for %q is %v", dbClone, ver)

	err = runner.Run(&types.Databases{
		GormDB:     gormDB,
		PostgresDB: pgPool,
	})
	if err != nil {
		return errors.Wrap(err, "migrations failed")
	}

	pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
	return nil
}
