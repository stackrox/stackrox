package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/tern/v2/migrate"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/postgreshelper"
	"github.com/stackrox/rox/migrator/runner"
	"github.com/stackrox/rox/migrator/types"
	migVer "github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"gorm.io/gorm"
)

func upgradeGORM(dbClone string) error {
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

	// If Postgres has no version, then we have no populated databases at all and thus don't
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

func upgradeTern(dbClone string) error {
	var pgPool postgres.DB
	var err error
	var migrationDir fs.FS

	// TODO(ROX-18005) Update to only use single DB when `central_previous` is no longer supported
	if pgconfig.IsExternalDatabase() {
		pgPool, _, err = postgreshelper.GetConnections()
	} else {
		pgPool, _, err = postgreshelper.Load(dbClone)
	}
	if err != nil {
		return errors.Wrap(err, "failed to connect to postgres DB")
	}
	// Close when needed
	defer postgreshelper.Close()

	ctx := sac.WithAllAccess(context.Background())
	conn, err := pgPool.Acquire(ctx)

	// Create a new migrator with schema_version as a version table
	migrator, err := migrate.NewMigrator(ctx, conn.Conn(), "schema_version")
	if err != nil {
		return errors.Wrap(err, "failed to initialize migrator")
	}

	if migrationsPath := env.TernMigrationsDir.Setting(); migrationsPath != "" {
		migrationDir = os.DirFS(migrationsPath)
	} else {
		// If not specified, try to find the directory assuming we are in the
		// stackrox repository
		rootDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not get working directory: %v\n", err)
		} else {
			for !strings.HasSuffix(rootDir, "stackrox") {
				rootDir = filepath.Dir(rootDir)
			}
		}

		migrationDir = os.DirFS(fmt.Sprintf("%s/migrations", rootDir))
	}

	err = migrator.LoadMigrations(migrationDir)
	if err != nil {
		return errors.Wrapf(err, "failed to load migrations at %s", migrationDir)
	}

	if len(migrator.Migrations) == 0 {
		return errors.Errorf("no migrations found at %s", migrationDir)
	}

	err = migrator.Migrate(ctx)
	if err != nil {
		return errors.Wrap(err, "migration has failed")
	}

	return nil
}
