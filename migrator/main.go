package main

import (
	"context"
	"net/http"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/compact"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/option"
	"github.com/stackrox/rox/migrator/postgreshelper"
	"github.com/stackrox/rox/migrator/replica/postgres"
	"github.com/stackrox/rox/migrator/replica/rocksdb"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/migrator/runner"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
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

	if features.PostgresDatastore.Enabled() {
		sourceMap, adminConfig, err := pgconfig.GetPostgresConfig()
		if err != nil {
			return errors.Wrap(err, "unable to get Postgres DB config.")
		}
		dbm := postgres.New(conf.Maintenance.ForceRollbackVersion, adminConfig, sourceMap)

		// Scan for database replicas
		err = dbm.Scan()
		if err != nil {
			return errors.Wrap(err, "fail to scan replicas")
		}

		// If we have database replicas we need to process them.  Otherwise, we can assume we are
		// starting from scratch and as such we will need to create the DB and apply schemas with Gorm.
		if len(dbm.ReplicaMap) > 0 {
			replica, _, err := dbm.GetReplicaToMigrate()
			log.WriteToStderrf("Replica to migrate => %q", replica)
			if err != nil {
				return err
			}

			// Run upgrades
			if err = upgrade(conf, replica); err != nil {
				return err
			}

			// Save the replica after upgrades have been processed
			if err = dbm.Persist(replica); err != nil {
				return err
			}
		} else {
			// No existing DB replica so need to create it.
			gormDB, err := postgreshelper.Load(conf)
			if err != nil {
				return errors.Wrap(err, "failed to connect to postgres DB")
			}
			pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
		}

		return nil
	}

	dbm := rocksdb.New(migrations.DBMountPath(), conf.Maintenance.ForceRollbackVersion)
	err := dbm.Scan()
	if err != nil {
		return errors.Wrap(err, "fail to scan replicas")
	}

	replica, replicaPath, err := dbm.GetReplicaToMigrate()
	if err != nil {
		return err
	}
	option.MigratorOptions.DBPathBase = replicaPath
	if err = upgrade(conf, ""); err != nil {
		return err
	}

	if err = dbm.Persist(replica); err != nil {
		return err
	}
	return nil
}

func upgrade(conf *config.Config, dbReplica string) error {
	if err := compact.Compact(conf); err != nil {
		log.WriteToStderrf("error compacting DB: %v", err)
	}

	boltDB, err := bolthelpers.Load()
	if err != nil {
		return errors.Wrap(err, "failed to open bolt DB")
	}
	if boltDB == nil {
		log.WriteToStderr("No DB found. Nothing to migrate...")
		return nil
	}

	rocksdb, err := rockshelper.New()
	if err != nil {
		return errors.Wrap(err, "failed to open rocksdb")
	}

	defer func() {
		if err := boltDB.Close(); err != nil {
			log.WriteToStderrf("Error closing DB: %v", err)
		}
		if rocksdb != nil {
			rocksdb.Close()
		}
	}()

	var gormDB *gorm.DB
	var pgPool *pgxpool.Pool
	if features.PostgresDatastore.Enabled() {
		gormDB, err = postgreshelper.Load(conf)
		if err != nil {
			return errors.Wrap(err, "failed to connect to postgres DB")
		}

		// We need to close the gorm connection when we are finished with it so the replica
		// movement can occur.
		sqlDB, err := gormDB.DB()
		if err != nil {
			log.WriteToStderrf("Error closing GormDB connection: %v", err)
		}
		defer sqlDB.Close()

		// We also need to get and close a connection to the active postgres database.
		_, pgConf, err := pgconfig.GetPostgresConfig()
		if err != nil {
			return errors.Wrap(err, "failed to connect to postgres DB")
		}

		pgPool = pgadmin.GetReplicaPool(pgConf, dbReplica)
		defer pgPool.Close()
	}

	err = runner.Run(&types.Databases{
		BoltDB:     boltDB,
		RocksDB:    rocksdb,
		GormDB:     gormDB,
		PostgresDB: pgPool,
	})
	if err != nil {
		return errors.Wrap(err, "migrations failed")
	}

	return nil
}
