package runner

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/migrator/log"
	"github.com/stackrox/stackrox/migrator/migrations"
	"github.com/stackrox/stackrox/migrator/types"
	pkgMigrations "github.com/stackrox/stackrox/pkg/migrations"
)

// Run runs the migrator.
func Run(databases *types.Databases) error {
	dbSeqNum, err := getCurrentSeqNum(databases)
	if err != nil {
		return errors.Wrap(err, "getting current seq num")
	}
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
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
			return fmt.Errorf("no migration found starting at %d", startingSeqNum)
		}
		err := migration.Run(databases)
		if err != nil {
			return errors.Wrapf(err, "error running migration starting at %d", startingSeqNum)
		}
		err = updateVersion(databases, &migration.VersionAfter)
		if err != nil {
			return errors.Wrapf(err, "failed to update version after migration %d", startingSeqNum)
		}
		log.WriteToStderrf("Successfully updated DB from version %d to %d", seqNum, migration.VersionAfter.GetSeqNum())
	}

	return nil
}
