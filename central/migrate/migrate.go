package migrate

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	cloneMgr "github.com/stackrox/rox/migrator/clone"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/version"
)

// Run executes the migration logic in-process using Central's existing DB
// pool. On the fast path (version matches), this returns immediately without
// creating any additional connections.
func Run(centralDB postgres.DB) error {
	log.WriteToStderrf("Run migrator with version: %s, DB sequence: %d",
		version.GetMainVersion(), migrations.CurrentDBVersionSeqNum())

	// Fast path: check if DB version matches using Central's existing pool.
	// This is also where the lazy pool establishes its first connection,
	// so we retry here until the database is reachable.
	dbSeqNum, err := readVersionSeqNum(centralDB)
	if err == nil {
		currSeqNum := migrations.CurrentDBVersionSeqNum()
		if dbSeqNum == currSeqNum {
			log.WriteToStderrf("DB is already at version %d, skipping migrations", dbSeqNum)
			return nil
		}
		log.WriteToStderrf("DB at version %d, binary at %d — running migrations", dbSeqNum, currSeqNum)
	}

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

// readVersionSeqNum attempts to read the DB version. It retries on connection
// errors (DB not yet reachable) but returns immediately on "database/table
// doesn't exist" which indicates a fresh install needing the full migration.
func readVersionSeqNum(db postgres.DB) (int, error) {
	var seqNum int
	for attempt := range 60 {
		ctx := sac.WithAllAccess(context.Background())
		err := db.QueryRow(ctx, "SELECT seqnum FROM versions LIMIT 1").Scan(&seqNum)
		if err == nil {
			return seqNum, nil
		}
		s := err.Error()
		if strings.Contains(s, "does not exist") || strings.Contains(s, "3D000") || strings.Contains(s, "42P01") {
			return 0, err
		}
		log.WriteToStderrf("waiting for database (attempt %d): %v", attempt+1, err)
		time.Sleep(5 * time.Second)
	}
	return 0, errors.New("timed out waiting for database")
}
