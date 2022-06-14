package main

import (
	"context"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/migrator/bolthelpers"
	"github.com/stackrox/stackrox/migrator/compact"
	"github.com/stackrox/stackrox/migrator/log"
	"github.com/stackrox/stackrox/migrator/option"
	"github.com/stackrox/stackrox/migrator/postgreshelper"
	"github.com/stackrox/stackrox/migrator/replica"
	"github.com/stackrox/stackrox/migrator/rockshelper"
	"github.com/stackrox/stackrox/migrator/runner"
	"github.com/stackrox/stackrox/migrator/types"
	"github.com/stackrox/stackrox/pkg/config"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/grpc/routes"
	"github.com/stackrox/stackrox/pkg/migrations"
	pkgSchema "github.com/stackrox/stackrox/pkg/postgres/schema"
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
	conf := config.GetConfig()
	if conf == nil {
		log.WriteToStderrf("cannot get central configuration. Skipping migrator")
		return nil
	}

	if conf.Maintenance.SafeMode {
		log.WriteToStderr("configuration has safe mode set. Skipping migrator")
		return nil
	}

	dbm, err := replica.Scan(migrations.DBMountPath(), conf.Maintenance.ForceRollbackVersion)
	if err != nil {
		return errors.Wrap(err, "fail to scan replicas")
	}

	replica, replicaPath, err := dbm.GetReplicaToMigrate()
	if err != nil {
		return err
	}
	option.MigratorOptions.DBPathBase = replicaPath
	if err = upgrade(conf); err != nil {
		return err
	}

	if features.PostgresDatastore.Enabled() {
		var gormDB *gorm.DB
		gormDB, err = postgreshelper.Load(conf)
		if err != nil {
			return errors.Wrap(err, "failed to connect to postgres DB")
		}
		pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
	}

	if err = dbm.Persist(replica); err != nil {
		return err
	}
	return nil
}

func upgrade(conf *config.Config) error {
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
	if features.PostgresDatastore.Enabled() {
		gormDB, err = postgreshelper.Load(conf)
		if err != nil {
			return errors.Wrap(err, "failed to connect to postgres DB")
		}
	}

	err = runner.Run(&types.Databases{
		BoltDB:  boltDB,
		RocksDB: rocksdb,
		GormDB:  gormDB,
	})
	if err != nil {
		return errors.Wrap(err, "migrations failed")
	}

	return nil
}
