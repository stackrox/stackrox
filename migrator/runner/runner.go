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
	switch {
	case dbSeqNum == 0:
		log.WriteToStderr("No version metadata; treating database as a fresh installation and skipping migrations.")
	case dbSeqNum > currSeqNum:
		log.WriteToStderrf("DB sequence number %d is greater than binary sequence %d; rolling back without migrations.", dbSeqNum, currSeqNum)
	case dbSeqNum == currSeqNum:
		log.WriteToStderrf("DB is up to date at version %d. Nothing to do here.", dbSeqNum)
	default:
		log.WriteToStderrf("Found DB at version %d, which is less than what we expect (%d). Running migrations...", dbSeqNum, currSeqNum)
		return runMigrations(databases, *source, migrations.Get)
	}

	// The caller publishes the target version only after applying all schemas.
	return nil
}

// UpdateToCurrentVersion updates the stored version to the current binary version
func UpdateToCurrentVersion(databases *types.Databases) error {
	minimum, err := pkgMigrations.MinimumSupportedDBVersionSeqNum()
	if err != nil {
		return errors.Wrap(err, "determining minimum supported database sequence")
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

	err = migVersion.UpdateVersionPostgres(ctx, databases.PostgresDB, currentVersion)
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
		return errors.Wrap(err, "determining minimum supported database sequence")
	}
	for seqNum := source.SeqNum; seqNum < pkgMigrations.CurrentDBVersionSeqNum(); seqNum++ {
		// Reset the context in case the previous migration replaced it.
		ctx := sac.WithAllAccess(context.Background())
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

		err = migVersion.UpdateVersionPostgres(ctx, databases.PostgresDB, migrationCheckpoint(source, int(migration.VersionAfter.GetSeqNum()), minimum))
		if err != nil {
			return errors.Wrapf(err, "failed to update version after migration %d", seqNum)
		}

		log.WriteToStderrf("Successfully updated DB from version %d to %d", seqNum, migration.VersionAfter.GetSeqNum())
	}

	return nil
}
