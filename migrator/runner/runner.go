package runner

import (
	"fmt"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
)

// Run runs the migrator.
func Run(boltDB *bolt.DB, badgerDB *badger.DB) error {
	dbSeqNum, err := getCurrentSeqNum(boltDB, badgerDB)
	if err != nil {
		return err
	}
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum
	if dbSeqNum > currSeqNum {
		return fmt.Errorf("DB sequence number %d is greater than the latest one we have (%d). This means "+
			"the migration binary is likely out of date", dbSeqNum, currSeqNum)
	}
	if dbSeqNum == currSeqNum {
		log.WriteToStderr("DB is up to date. Nothing to do here.")
		return nil
	}
	log.WriteToStderr("Found DB at version %d, which is less than what we expect (%d). Running migrations...",
		dbSeqNum, currSeqNum)
	return runMigrations(boltDB, badgerDB, dbSeqNum)
}

func runMigrations(boltDB *bolt.DB, badgerDB *badger.DB, startingSeqNum int) error {
	for seqNum := startingSeqNum; seqNum < pkgMigrations.CurrentDBVersionSeqNum; seqNum++ {
		migration, ok := migrations.Get(seqNum)
		if !ok {
			return fmt.Errorf("no migration found starting at %d", startingSeqNum)
		}
		err := migration.Run(boltDB, badgerDB)
		if err != nil {
			return errors.Wrapf(err, "error running migration starting at %d", startingSeqNum)
		}
		err = updateVersion(boltDB, badgerDB, &migration.VersionAfter)
		if err != nil {
			return errors.Wrapf(err, "failed to update version after migration %d", startingSeqNum)
		}
		log.WriteToStderr("Successfully updated DB from version %d to %d", seqNum, migration.VersionAfter.GetSeqNum())
	}
	return nil
}
