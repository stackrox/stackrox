package runner

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/set"
)

var skipMigrationMap = set.NewIntSet()

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
		return fmt.Errorf("DB sequence number %d is greater than the latest one we have (%d). This means "+
			"the migration binary is likely out of date", dbSeqNum, currSeqNum)
	}
	if dbSeqNum != currSeqNum {
		log.WriteToStderrf("Found DB at version %d, which is less than what we expect (%d). Running migrations...", dbSeqNum, currSeqNum)
		if err := runMigrations(databases, dbSeqNum); err != nil {
			return err
		}
	} else {
		log.WriteToStderrf("DB is up to date at version %d. Nothing to do here.", dbSeqNum)
	}

	return nil
}

func runMigrations(databases *types.Databases, startingSeqNum int) error {
	for seqNum := startingSeqNum; seqNum < pkgMigrations.CurrentDBVersionSeqNum(); seqNum++ {
		migration, ok := migrations.Get(seqNum)
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

		err := updateVersion(databases, migration.VersionAfter)
		if err != nil {
			return errors.Wrapf(err, "failed to update version after migration %d", seqNum)
		}
		log.WriteToStderrf("Successfully updated DB from version %d to %d", seqNum, migration.VersionAfter.GetSeqNum())
	}

	return nil
}
