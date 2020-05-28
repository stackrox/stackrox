package runner

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/env"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
)

func runRocksDBMigrationIfNecessary(databases *types.Databases) error {
	// If we are not using RocksDB or BadgerDB is nil, which means a migration already occurred
	// then return
	if !env.RocksDB.BooleanSetting() || databases.BadgerDB == nil {
		return nil
	}

	// Migrate BadgerDB -> RocksDB
	// If Badger was opened, then the migration still needs to be done
	if err := rocksdbmigration.Migrate(databases); err != nil {
		return errors.Wrap(err, "migrating to RocksDB")
	}
	// Update RocksDB version to mark successful migration
	migration, ok := migrations.Get(pkgMigrations.CurrentDBVersionSeqNum - 1)
	if !ok {
		return errors.Errorf("migration at current db version %d - 1 must exist", pkgMigrations.CurrentDBVersionSeqNum)
	}
	versionBytes, err := proto.Marshal(&migration.VersionAfter)
	if err != nil {
		return errors.Wrap(err, "marshalling version")
	}
	if err := updateRocksDB(databases.RocksDB, versionBytes); err != nil {
		return err
	}
	return nil
}

// Run runs the migrator.
func Run(databases *types.Databases) error {
	dbSeqNum, err := getCurrentSeqNum(databases)
	if err != nil {
		return errors.Wrap(err, "getting current seq num")
	}
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum
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
		log.WriteToStderr("DB is up to date. Nothing to do here.")
	}

	return runRocksDBMigrationIfNecessary(databases)
}

func runMigrations(databases *types.Databases, startingSeqNum int) error {
	for seqNum := startingSeqNum; seqNum < pkgMigrations.CurrentDBVersionSeqNum; seqNum++ {
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
