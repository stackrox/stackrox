package m40to41

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

var (
	versionBucketName = []byte("version")

	migration = types.Migration{
		StartingSeqNum: 40,
		VersionAfter:   storage.Version{SeqNum: 41},
		Run:            rocksDBMigration,
	}
)

// Get the DB version number from RocksDB. If there is no entry,
// return 0
func getCurrentSeqNumRocksDB(db *gorocksdb.DB) (int, error) {
	var version storage.Version

	opts := gorocksdb.NewDefaultReadOptions()
	defer opts.Destroy()
	slice, err := db.Get(opts, versionBucketName)
	if err != nil || !slice.Exists() {
		return 0, err
	}
	defer slice.Free()
	if err := proto.Unmarshal(slice.Data(), &version); err != nil {
		return 0, err
	}
	return int(version.GetSeqNum()), nil
}

func rocksDBMigration(databases *types.Databases) error {
	rocksDBSeqNum, err := getCurrentSeqNumRocksDB(databases.RocksDB)
	if err != nil {
		return errors.Wrap(err, "failed to fetch sequence number from rocksdb")
	}
	// RocksDB migration was previously executed so no need to execute it again
	if rocksDBSeqNum != 0 {
		return nil
	}

	return rocksdbmigration.Migrate(databases)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
