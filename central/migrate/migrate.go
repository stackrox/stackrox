package migrate

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	cloneMgr "github.com/stackrox/rox/migrator/clone"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/version"
)

// Run executes the migration logic in-process. This replaces the separate
// migrator binary invocation from start-central.sh.
func Run() error {
	log.WriteToStderrf("Run migrator with version: %s, DB sequence: %d",
		version.GetMainVersion(), migrations.CurrentDBVersionSeqNum())

	conf := config.GetConfig()
	if conf == nil {
		log.WriteToStderr("cannot get central configuration. Skipping migrator")
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

	if !pgconfig.IsExternalDatabase() {
		if err := ensureDatabaseExists(); err != nil {
			return err
		}
	}

	sourceMap, adminConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return errors.Wrap(err, "unable to get Postgres DB config")
	}

	dbm := cloneMgr.NewPostgres(rollbackVersion, adminConfig, sourceMap)

	if err := dbm.Scan(); err != nil {
		return errors.Wrap(err, "failed to scan clones")
	}

	pgClone, err := dbm.GetCloneToMigrate()
	if err != nil {
		return errors.Wrap(err, "failed to get clone to migrate")
	}
	log.WriteToStderrf("Clone to Migrate %q", pgClone)

	if err := upgrade(pgClone); err != nil {
		return err
	}

	return dbm.Persist(pgClone)
}

func ensureDatabaseExists() error {
	sourceMap, adminConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return err
	}
	return retry.WithRetry(func() error {
		log.WriteToStderrf("checking if the database %q exists", pgconfig.GetActiveDB())
		exists, err := pgadmin.CheckIfDBExists(adminConfig, pgconfig.GetActiveDB())
		if err != nil {
			return err
		}
		if !exists {
			return pgadmin.CreateDB(sourceMap, adminConfig, pgadmin.EmptyDB, pgconfig.GetActiveDB())
		}
		return nil
	}, retry.Tries(60), retry.BetweenAttempts(func(_ int) {
		time.Sleep(5 * time.Second)
	}))
}
