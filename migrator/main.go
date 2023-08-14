package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	cloneMgr "github.com/stackrox/rox/migrator/clone"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/option"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/version"
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
	log.WriteToStderrf("Run migrator.run() with version: %s, DB sequence: %d", version.GetMainVersion(), migrations.CurrentDBVersionSeqNum())
	conf := config.GetConfig()
	if conf == nil {
		log.WriteToStderrf("cannot get central configuration. Skipping migrator")
		return nil
	}

	if conf.Maintenance.SafeMode {
		log.WriteToStderr("configuration has safe mode set. Skipping migrator")
		return nil
	}

	rollbackVersion := strings.TrimSpace(conf.Maintenance.ForceRollbackVersion)
	if rollbackVersion != "" {
		log.WriteToStderrf("conf.Maintenance.ForceRollbackVersion: %s", rollbackVersion)
	}

	// If using internal database, ensure the default database (`central_active`) exists
	if !pgconfig.IsExternalDatabase() {
		if err := ensureDatabaseExists(); err != nil {
			return err
		}
	}

	// Create the clone manager
	sourceMap, adminConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return errors.Wrap(err, "unable to get Postgres DB config")
	}

	dbm := cloneMgr.NewPostgres(migrations.DBMountPath(), rollbackVersion, adminConfig, sourceMap)

	err = dbm.Scan()
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

func dbCheck(source map[string]string, adminConfig *postgres.Config) error {
	// Create the central database if necessary
	log.WriteToStderrf("checking if the database %q exists", pgconfig.GetActiveDB())
	exists, err := pgadmin.CheckIfDBExists(adminConfig, pgconfig.GetActiveDB())
	if err != nil {
		log.WriteToStderrf("Could not check for central database: %v", err)
		return err
	}
	if !exists {
		err = pgadmin.CreateDB(source, adminConfig, pgadmin.EmptyDB, pgconfig.GetActiveDB())
		if err != nil {
			log.WriteToStderrf("Could not create central database: %v", err)
			return err
		}
	}
	return nil
}

func ensureDatabaseExists() error {
	sourceMap, adminConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return err
	}

	if !pgconfig.IsExternalDatabase() {
		return retry.WithRetry(func() error {
			return dbCheck(sourceMap, adminConfig)
		}, retry.Tries(10), retry.BetweenAttempts(func(_ int) {
			time.Sleep(5 * time.Second)
		}))
	}
	return nil
}
