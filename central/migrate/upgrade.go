package migrate

import (
	"context"

	"github.com/pkg/errors"
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
	defer postgreshelper.Close()

	return upgradeAcquireLock(pgPool, gormDB, dbClone)
}

func upgradeAcquireLock(pgPool postgres.DB, gormDB *gorm.DB, dbClone string) error {
	ctx := sac.WithAllAccess(context.Background())
	ver, err := migVer.ReadVersionGormDB(ctx, gormDB)
	if err != nil {
		return errors.Wrap(err, "failed to get version from the database")
	}

	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()

	acquired, release, err := lock.TryAcquireMigrationLock(ctx, pgPool)
	if err != nil {
		return errors.Wrap(err, "failed to try migration advisory lock")
	}

	if acquired {
		defer release()
		return upgradeWithLock(ctx, pgPool, gormDB, dbClone)
	}

	log.WriteToStderrf(
		"Migration lock held by another instance. DB seqnum = %d, binary seqnum %d.",
		ver.SeqNum, currSeqNum,
	)

	switch {
	case currSeqNum > ver.SeqNum:
		return errors.Errorf("failed to upgrade DB from %d to %d, could not acquire migration lock.", ver.SeqNum, currSeqNum)
	case currSeqNum < ver.SeqNum:
		log.WriteToStderrf("Old version pod proceeding without migrations (DB seqnum %d > binary seqnum %d).", ver.SeqNum, currSeqNum)
	default:
		log.WriteToStderr("DB version matches current version, skipping migrations.")
	}

	return nil
}

func upgradeWithLock(ctx context.Context, pgPool postgres.DB, gormDB *gorm.DB, dbClone string) error {
	ver, err := migVer.ReadVersionGormDB(ctx, gormDB)
	if err != nil {
		return errors.Wrap(err, "failed to re-read version from the database after acquiring lock")
	}

	if ver.SeqNum == 0 && ver.MainVersion == "0" {
		log.WriteToStderr("Fresh install of the database. There is no data to migrate...")
		pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
		migVer.SetCurrentVersion(ctx, gormDB)
		return nil
	}
	log.WriteToStderrf("version for %q is %v", dbClone, ver)

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
