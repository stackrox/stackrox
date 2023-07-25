//go:build amd64

package main

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/compact"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/option"
	"github.com/stackrox/rox/migrator/postgreshelper"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/migrator/runner"
	"github.com/stackrox/rox/migrator/types"
	migVer "github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/tecbot/gorocksdb"
	"go.etcd.io/bbolt"
	"gorm.io/gorm"
)

func upgrade(conf *config.Config, dbClone string, processBoth bool) error {
	err := compact.Compact(conf)
	if err != nil {
		log.WriteToStderrf("error compacting DB: %v", err)
	}

	var pkgRocksDB *rocksdb.RocksDB
	var rocks *gorocksdb.DB
	var boltDB *bbolt.DB

	// We need to pass Rocks to the runner if we are in Rocks mode OR
	// if we need to processBoth for the purpose of migrating Rocks to Postgres
	if processBoth {
		boltDB, err = bolthelpers.Load()
		if err != nil {
			return errors.Wrap(err, "failed to open bolt DB")
		}
		if boltDB == nil {
			log.WriteToStderr("No legacy DB found. Nothing to migrate...")
		} else {
			pkgRocksDB = rockshelper.GetRocksDB()
			rocks = pkgRocksDB.DB
			defer func() {
				if err := boltDB.Close(); err != nil {
					log.WriteToStderrf("Error closing DB: %v", err)
				}
				if rocks != nil {
					rocks.Close()
				}
			}()
		}
	}

	var gormDB *gorm.DB
	var pgPool postgres.DB
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
	if ver.SeqNum == 0 && ver.MainVersion == "0" && (!processBoth || boltDB == nil) {
		log.WriteToStderr("Fresh install of the database. There is no data to migrate...")
		pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
		return nil
	}
	log.WriteToStderrf("version for %q is %v", dbClone, ver)

	err = runner.Run(&types.Databases{
		BoltDB:     boltDB,
		RocksDB:    rocks,
		GormDB:     gormDB,
		PostgresDB: pgPool,
		PkgRocksDB: pkgRocksDB,
	})
	if err != nil {
		return errors.Wrap(err, "migrations failed")
	}

	// If we need to process Rocks and Postgres we used Rocks to populate Postgres.  As such we still need
	// to update the current version of Rocks.  Central takes care of that for active databases, but since
	// Rocks will not be the active database, we need to do that as part of the migrations.
	if processBoth {
		// Update last associated software version on DBs.
		migrations.SealLegacyDB(option.MigratorOptions.DBPathBase)
	}

	if gormDB != nil {
		pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
	}

	return nil
}
