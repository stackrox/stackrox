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
	migVersion "github.com/stackrox/rox/migrator/version"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/version"
)

var (
	skipMigrationMap = set.NewIntSet()
)

func init() {
	env := os.Getenv("ROX_SKIP_MIGRATIONS")
	if env == "" {
		return
	}
	for skipped := range strings.SplitSeq(env, ",") {
		migration, err := strconv.Atoi(strings.TrimSpace(skipped))
		if err != nil {
			log.WriteToStderrf("could not parse %v. Not skipping", skipped)
			continue
		}
		skipMigrationMap.Add(migration)
	}
}

// Run runs the migrator. It reads the current DB version and runs all
// necessary migrations to bring the DB up to the current binary version.
func Run(databases *types.Databases) error {
	log.WriteToStderrf("In runner.Run")

	source, err := migVersion.ReadVersionGormDB(sac.WithAllAccess(context.Background()), databases.GormDB)
	if err != nil {
		return errors.Wrap(err, "getting current seq num")
	}
	dbSeqNum := source.SeqNum
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
		if err := runMigrations(databases, *source, migrations.Get); err != nil {
			return err
		}
	} else {
		log.WriteToStderrf("DB is up to date at version %d. Nothing to do here.", dbSeqNum)
	}

	// The caller publishes the target version only after applying all schemas.
	return nil
}

// Preflight verifies the entire migration path before the first migration runs.
// An unsafe override cannot recreate migrations that have been pruned.
func Preflight(startingSeqNum int) error {
	for seq := startingSeqNum; seq < pkgMigrations.CurrentDBVersionSeqNum(); seq++ {
		if _, ok := migrations.Get(seq); !ok {
			return &pkgMigrations.CompatibilityError{Message: fmt.Sprintf("Central upgrade blocked: no migration found starting at sequence %d. Use an intermediate Central release containing the missing migrations; the unsafe override cannot bypass this check.", seq)}
		}
	}
	return nil
}

// UpdateToCurrentVersion updates the stored version to the current binary version
func UpdateToCurrentVersion(databases *types.Databases) error {
	minimum, err := pkgMigrations.MinimumSupportedDBVersionSeqNum()
	if err != nil {
		return err
	}
	ctx := sac.WithAllAccess(context.Background())
	source, err := migVersion.ReadVersionGormDB(ctx, databases.GormDB)
	if err != nil {
		return errors.Wrap(err, "reading version before publishing completed migration")
	}
	currentVersion := &versionStorage.Version{
		SeqNum:        int32(pkgMigrations.CurrentDBVersionSeqNum()),
		Version:       version.GetMainVersion(),
		MinSeqNum:     int32(max(source.MinimumSeqNum, minimum)),
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}

	err = updateVersion(ctx, databases, currentVersion)
	if err != nil {
		return errors.Wrapf(err, "failed to update version after migrations %d", currentVersion.GetSeqNum())
	}
	return nil
}

func migrationCheckpoint(source pkgMigrations.MigrationVersion, sequence, minimum int) *versionStorage.Version {
	return &versionStorage.Version{
		SeqNum:        int32(sequence),
		Version:       source.MainVersion,
		MinSeqNum:     int32(max(source.MinimumSeqNum, minimum)),
		LastPersisted: protoconv.ConvertTimeToTimestampOrNil(source.LastPersisted),
	}
}

func runMigrations(databases *types.Databases, source pkgMigrations.MigrationVersion, lookup func(int) (types.Migration, bool)) error {
	minimum, err := pkgMigrations.MinimumSupportedDBVersionSeqNum()
	if err != nil {
		return err
	}
	for seqNum := source.SeqNum; seqNum < pkgMigrations.CurrentDBVersionSeqNum(); seqNum++ {
		// Add an outer transaction so migrations can be wrapped in a transaction.
		ctx := sac.WithAllAccess(context.Background())

		// Set the context with the databases so the wrapped transaction can be used
		databases.DBCtx = ctx

		migration, ok := lookup(seqNum)
		if !ok {
			return fmt.Errorf("no migration found starting at %d", seqNum)
		}

		if skipMigrationMap.Contains(seqNum) {
			log.WriteToStderrf("Skipping migration %d based on environment variable", seqNum)
		} else {
			err := migration.Run(databases)
			if err != nil {
				return errors.Wrapf(err, "error running migration starting at %d", seqNum)
			}
		}

		tx, err := databases.PostgresDB.Begin(ctx)
		if err != nil {
			return err
		}
		ctx = pgPkg.ContextWithTx(ctx, tx)

		err = updateVersion(ctx, databases, migrationCheckpoint(source, int(migration.VersionAfter.GetSeqNum()), minimum))
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
