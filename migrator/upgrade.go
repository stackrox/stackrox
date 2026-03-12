package main

import (
	"context"

	"github.com/pkg/errors"
	versionStorage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/lock"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/postgreshelper"
	"github.com/stackrox/rox/migrator/runner"
	"github.com/stackrox/rox/migrator/types"
	migVer "github.com/stackrox/rox/migrator/version"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"gorm.io/gorm"
)

func upgrade(dbClone string) error {
	var gormDB *gorm.DB
	var pgPool postgres.DB
	var err error
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

	return upgradeAcquireLock(pgPool, gormDB, dbClone)
}

// upgradeAcquireLock runs the logic to acquire the migration lock
// and run the upgrade if successfully locked
func upgradeAcquireLock(pgPool postgres.DB, gormDB *gorm.DB, dbClone string) error {
	ctx := sac.WithAllAccess(context.Background())
	ver, err := migVer.ReadVersionGormDB(ctx, gormDB)
	if err != nil {
		return errors.Wrap(err, "failed to get version from the database")
	}

	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()

	// Try to acquire the advisory lock without blocking.
	acquired, release, err := lock.TryAcquireMigrationLock(ctx, pgPool)
	if err != nil {
		return errors.Wrap(err, "failed to try migration advisory lock")
	}

	if acquired {
		defer release()
		return upgradeWithLock(ctx, pgPool, gormDB, dbClone)
	}

	// At this point another instance holds the lock (likely running migrations).
	log.WriteToStderrf(
		"Migration lock held by another instance. DB seqnum = %d, binary seqnum %d.",
		ver.SeqNum, currSeqNum,
	)

	switch {
	case currSeqNum > ver.SeqNum:
		// This is a new version pod trying to upgrade the DB
		// Fail fast, to restart the container and try acquiring lock again.
		return errors.Errorf("failed to upgrade DB from %d to %d, could not acquire migration lock.", ver.SeqNum, currSeqNum)
	case currSeqNum < ver.SeqNum:
		// This is the old pod during a rolling upgrade. Write a rollback marker,
		// so the next lock holder can reset the seqnum if the upgrade fails.
		log.WriteToStderrf("Writing rollback marker to %d and proceeding without migrations.", currSeqNum)
		if err := migVer.WriteRollbackSeqNum(gormDB, currSeqNum); err != nil {
			return errors.Wrap(err, "failed to write rollback marker")
		}
	default:
		// DB version matches current version, but couldn't acquire lock
		log.WriteToStderrf("DB version matches current version, skipping migrations.")
	}

	return nil
}

// upgradeWithLock runs migrations and schema application while holding the
// advisory lock. It checks the rollback marker and honors it if a rollback
// occurred while the lock was not held.
func upgradeWithLock(ctx context.Context, pgPool postgres.DB, gormDB *gorm.DB, dbClone string) error {
	// Re-read the version after acquiring the lock. Another instance may have
	// completed migrations between the first read and the acquisition of the lock.
	ver, err := migVer.ReadVersionGormDB(ctx, gormDB)
	if err != nil {
		return errors.Wrap(err, "failed to re-read version from the database after acquiring lock")
	}

	// If Postgres has no version, then we have no populated databases at all and thus don't
	// need to migrate
	if ver.SeqNum == 0 && ver.MainVersion == "0" {
		log.WriteToStderr("Fresh install of the database. There is no data to migrate...")
		pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
		migVer.SetCurrentVersionGormDB(ctx, gormDB)
		return nil
	}
	log.WriteToStderrf("version for %q is %v", dbClone, ver)

	checkAndResetRollbackMarker(ctx, gormDB, ver)

	databases := &types.Databases{
		GormDB:     gormDB,
		PostgresDB: pgPool,
	}

	if err := runner.Run(databases); err != nil {
		return errors.Wrap(err, "migrations failed")
	}

	pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
	return nil
}

func checkAndResetRollbackMarker(ctx context.Context, gormDB *gorm.DB, dbVer *pkgMigrations.MigrationVersion) {
	if dbVer.RollbackSeqNum != 0 && dbVer.SeqNum > dbVer.RollbackSeqNum {
		// restart of a old version pod happened while a migration was in progress:
		// the old pod wrote a marker for deferred rollback before it exited.
		log.WriteToStderrf("Rollback marker found: rollbackSeqNum = %d, dbSeqNum = %d. "+
			"Resetting DB version to marker.", dbVer.RollbackSeqNum, dbVer.SeqNum)

		startMigFromVer := &versionStorage.Version{
			SeqNum:         int32(dbVer.RollbackSeqNum),
			Version:        dbVer.MainVersion,
			MinSeqNum:      int32(dbVer.MinimumSeqNum),
			RollbackSeqNum: 0,
		}

		migVer.SetVersionGormDB(ctx, gormDB, startMigFromVer, false)
	}
}
