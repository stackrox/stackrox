package m42to43

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
)

var (
	bucketName = []byte("apiTokens")

	migration = types.Migration{
		StartingSeqNum: 42,
		VersionAfter:   storage.Version{SeqNum: 43},
		Run:            rocksDBMigration,
	}
)

func rocksDBMigration(databases *types.Databases) error {
	log.WriteToStderr("Migrating API Tokens to RocksDB")
	count, err := rocksdbmigration.MigrateBoltBucket(databases.BoltDB, databases.RocksDB, []byte(bucketName))
	if err != nil {
		return err
	}
	log.WriteToStderrf("Successfully wrote %d API Tokens to RocksDB", count)
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
