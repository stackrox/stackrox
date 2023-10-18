package runner

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	versionStorage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
)

var (
	skipMigrationMap = set.NewIntSet()
)

func init() {
	env := os.Getenv("ROX_SKIP_MIGRATIONS")
	if env == "" {
		return
	}
	for _, skipped := range strings.Split(env, ",") {
		migration, err := strconv.Atoi(strings.TrimSpace(skipped))
		if err != nil {
			log.WriteToStderrf("could not parse %v. Not skipping", skipped)
			continue
		}
		skipMigrationMap.Add(migration)
	}
}

// Run runs the migrator.
func Run(databases *types.Databases) error {
	log.WriteToStderrf("In runner.Run")

	// If Rocks and Bolt are passed into this function when Postgres is enabled, that means
	// we are in a state where we need to migrate Rocks to Postgres.  In this case the Rocks
	// sequence number will be returned and used to drive the migrations
	dbSeqNum, err := getCurrentSeqNum(databases)
	if err != nil {
		return errors.Wrap(err, "getting current seq num")
	}
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	if dbSeqNum == 0 {
		log.WriteToStderr("Sequence number of 0 means starting fresh, no migrations to execute")
		return nil
	}
	if dbSeqNum > currSeqNum {
		log.WriteToStderrf("DB sequence number %d is greater than the latest one we have (%d). This means "+
			"we are in a rollback.", dbSeqNum, currSeqNum)
	}
	if dbSeqNum < currSeqNum {
		log.WriteToStderrf("Found DB at version %d, which is less than what we expect (%d). Running migrations...", dbSeqNum, currSeqNum)
		if err := runMigrations(databases, dbSeqNum); err != nil {
			return err
		}
	} else {
		log.WriteToStderrf("DB is up to date at version %d. Nothing to do here.", dbSeqNum)
	}

	// Make sure version is up to date after migrations to ensure latest version schema is used in the event
	// there are no migrations executed.
	currentVersion := &versionStorage.Version{SeqNum: int32(pkgMigrations.CurrentDBVersionSeqNum())}
	ctx := sac.WithAllAccess(context.Background())
	err = updateVersion(ctx, databases, currentVersion)
	if err != nil {
		return errors.Wrapf(err, "failed to update version after migrations %d", currentVersion.SeqNum)
	}

	return nil
}

func runMigrations(databases *types.Databases, startingSeqNum int) error {
	for seqNum := startingSeqNum; seqNum < pkgMigrations.CurrentDBVersionSeqNum(); seqNum++ {
		// Add an outer transaction so migrations can be wrapped in a transaction.
		ctx := sac.WithAllAccess(context.Background())

		tx, err := databases.PostgresDB.Begin(ctx)
		if err != nil {
			return err
		}
		ctx = pgPkg.ContextWithTx(ctx, tx)

		// Set the context with the databases so the wrapped transaction can be used
		databases.DBCtx = ctx

		migration, ok := migrations.Get(seqNum)
		if !ok {
			return fmt.Errorf("no migration found starting at %d", seqNum)
		}

		if skipMigrationMap.Contains(seqNum) {
			log.WriteToStderrf("Skipping migration %d based on environment variable", seqNum)
		} else {
			err := migration.Run(databases)
			if err != nil {
				return wrapRollback(ctx, tx, errors.Wrapf(err, "error running migration starting at %d", seqNum))
			}
		}

		err = updateVersion(ctx, databases, migration.VersionAfter)
		if err != nil {
			return wrapRollback(ctx, tx, errors.Wrapf(err, "failed to update version after migration %d", seqNum))
		}

		err = tx.Commit(ctx)
		if err != nil {
			return wrapRollback(ctx, tx, errors.Wrapf(err, "unable to commit migration starting at %d", seqNum))
		}
		log.WriteToStderrf("Successfully updated DB from version %d to %d", seqNum, migration.VersionAfter.GetSeqNum())
	}

	return nil
}

func wrapRollback(ctx context.Context, tx *pgPkg.Tx, err error) error {
	rollbackErr := tx.Rollback(ctx)
	if rollbackErr != nil {
		return errors.Wrapf(rollbackErr, "rolling back due to err: %v", err)
	}
	return err
}
