package main

import (
	"context"
	"net/http"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/bolthelpers"
	cloneMgr "github.com/stackrox/rox/migrator/clone"
	"github.com/stackrox/rox/migrator/compact"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/option"
	"github.com/stackrox/rox/migrator/postgreshelper"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/migrator/runner"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/tecbot/gorocksdb"
	"go.etcd.io/bbolt"
	"gorm.io/gorm"
)

func main() {
	startProfilingServer()
	if err := run(); err != nil {
		log.WriteToStderrf("Migrator failed: %s", err)
		os.Exit(1)
	}
}

func startProfilingServer() {
	handler := http.NewServeMux()
	for path, debugHandler := range routes.DebugRoutes {
		handler.Handle(path, debugHandler)
	}
	srv := &http.Server{Addr: ":6060", Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.WriteToStderrf("Closing profiling server: %v", err)
		}
	}()
}

func run() error {
	log.WriteToStderr("In migrator.run()")
	conf := config.GetConfig()
	if conf == nil {
		log.WriteToStderrf("cannot get central configuration. Skipping migrator")
		return nil
	}

	if conf.Maintenance.SafeMode {
		log.WriteToStderr("configuration has safe mode set. Skipping migrator")
		return nil
	}

	var dbm cloneMgr.DBCloneManager
	// Create the clone manager
	if features.PostgresDatastore.Enabled() {
		sourceMap, adminConfig, err := pgconfig.GetPostgresConfig()
		if err != nil {
			return errors.Wrap(err, "unable to get Postgres DB config")
		}
		dbm = cloneMgr.NewPostgres(migrations.DBMountPath(), conf.Maintenance.ForceRollbackVersion, adminConfig, sourceMap)
	} else {
		dbm = cloneMgr.New(migrations.DBMountPath(), conf.Maintenance.ForceRollbackVersion)
	}

	err := dbm.Scan()
	if err != nil {
		return errors.Wrap(err, "failed to scan clones")
	}

	// Get the clone we are migrating
	clone, clonePath, pgClone, err := dbm.GetCloneToMigrate()
	if err != nil {
		return errors.Wrap(err, "failed to get clone to migrate")
	}
	log.WriteToStderrf("Clone to Migrate %q, %q", clone, pgClone)

	// Set the path to Rocks if it exists.
	if clonePath != "" {
		option.MigratorOptions.DBPathBase = clonePath
	}

	// If GetCloneToMigrate returns Rocks and Postgres clones that means we need
	// to migrate Rocks->Postgres.  Otherwise, we need to process Rocks in Rocks mode and
	// Postgres in Postgres mode.
	processBoth := clone != "" && pgClone != ""

	err = upgrade(conf, pgClone, processBoth)
	if err != nil {
		return err
	}

	if err = dbm.Persist(clone, pgClone, processBoth); err != nil {
		return err
	}

	return nil
}

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
	if !features.PostgresDatastore.Enabled() || processBoth {
		boltDB, err = bolthelpers.Load()
		if err != nil {
			return errors.Wrap(err, "failed to open bolt DB")
		}
		if boltDB == nil {
			log.WriteToStderr("No legacy DB found. Nothing to migrate...")
		} else {
			pkgRocksDB, err = rockshelper.New()
			if err != nil {
				return errors.Wrap(err, "failed to open rocksdb")
			}

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
	var pgPool *pgxpool.Pool
	if features.PostgresDatastore.Enabled() {
		pgPool, gormDB, err = postgreshelper.Load(conf, dbClone)
		if err != nil {
			return errors.Wrap(err, "failed to connect to postgres DB")
		}
		// Close when needed
		defer postgreshelper.Close()

		ver, err := migrations.ReadVersionPostgres(pgPool)
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

		// We need to migrate so turn off foreign key constraints
		// Now that migrations are complete, turn the constraints back on
		gormConfig := gormDB.Config
		gormConfig.DisableForeignKeyConstraintWhenMigrating = true
		err = gormDB.Apply(gormConfig)
		if err != nil {
			return errors.Wrap(err, "failed to turn off foreign key constraints")
		}

		log.WriteToStderrf("version for %q is %v", dbClone, ver)
	}

	if boltDB == nil && !features.PostgresDatastore.Enabled() {
		log.WriteToStderr("No DB found. Nothing to migrate...")
		return nil
	}

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
		migrations.SetCurrent(option.MigratorOptions.DBPathBase)
	}

	if gormDB != nil {
		// Now that migrations are complete, turn the constraints back on.  It is assumed the migrations
		// removed any rows that violated constraints.
		gormConfig := gormDB.Config
		gormConfig.DisableForeignKeyConstraintWhenMigrating = false
		err = gormDB.Apply(gormConfig)
		if err != nil {
			return errors.Wrap(err, "failed to turn on foreign key constraints")
		}
		pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
	}

	return nil
}
