package rocksdbmigration

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/types"
)

const (
	batchSize = 500
)

// Migrate migrates all of BadgerDB and parts of BoltDB to RocksDB
func Migrate(databases *types.Databases) error {
	log.WriteToStderr("Starting to migrate from BadgerDB to RocksDB")
	if err := migrateBadger(databases); err != nil {
		return errors.Wrap(err, "migrating badger -> rocksdb")
	}
	log.WriteToStderr("Successfully migrated BadgerDB to RocksDB")
	if err := migrateBolt(databases); err != nil {
		return errors.Wrap(err, "migrating bolt -> rocksdb")
	}
	log.WriteToStderr("Successfully migrated from BoltDB to RocksDB")
	return nil
}
