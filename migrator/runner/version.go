package runner

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
)

var (
	versionBucketName = []byte("version")
	versionKey        = []byte("\x00")
	ctx               = sac.WithAllAccess(context.Background())
)

// getCurrentSeqNumBolt returns the current seq-num found in the bolt DB.
// A returned value of 0 means that the version bucket was not found in the DB;
// this special value is only returned when we're upgrading from a version pre-2.4.
func getCurrentSeqNumBolt(db *bolt.DB) (int, error) {
	bucketExists, err := bolthelpers.BucketExists(db, versionBucketName)
	if err != nil {
		return 0, errors.Wrap(err, "checking for version bucket existence")
	}
	if !bucketExists {
		return 0, nil
	}
	versionBucket := bolthelpers.TopLevelRef(db, versionBucketName)
	versionBytes, err := bolthelpers.RetrieveElementAtKey(versionBucket, versionKey)
	if err != nil {
		return 0, errors.Wrap(err, "failed to retrieve version")
	}
	if versionBytes == nil {
		return 0, errors.New("INVALID STATE: a version bucket existed, but no version was found")
	}
	version := new(storage.Version)
	err = proto.Unmarshal(versionBytes, version)
	if err != nil {
		return 0, errors.Wrap(err, "unmarshaling version proto")
	}
	return int(version.GetSeqNum()), nil
}

// getCurrentSeqNumRocksDB returns the current seq-num found in the rocks DB.
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

func getCurrentSeqNumPostgres(databases *types.Databases) (int, error) {
	ver, err := version.ReadVersionGormDB(ctx, databases.GormDB)
	if err != nil {
		return 0, errors.Wrap(err, "getting current postgres sequence number")
	}

	return ver.SeqNum, nil
}
func getCurrentSeqNum(databases *types.Databases) (int, error) {
	var currentSeqNumPostgres int
	var err error
	// If Rocks and Bolt are passed into this function when Postgres is enabled, that means
	// we are in a state where we need to migrate Rocks to Postgres.  In this case the Rocks
	// sequence number will take precedence and drive the migrations
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		currentSeqNumPostgres, err = getCurrentSeqNumPostgres(databases)
		if err != nil {
			return 0, err
		}
		if databases.RocksDB == nil && databases.BoltDB == nil {
			return currentSeqNumPostgres, nil
		}
	}

	// Legacy databases should be present at this point.
	if databases.RocksDB == nil || databases.BoltDB == nil {
		return 0, errors.New("legacy databases do not not exist")
	}

	boltSeqNum, err := getCurrentSeqNumBolt(databases.BoltDB)
	if err != nil {
		return 0, errors.Wrap(err, "getting current bolt sequence number")
	}

	writeHeavySeqNum, err := getCurrentSeqNumRocksDB(databases.RocksDB)
	if err != nil {
		return 0, errors.Wrap(err, "getting current rocksdb sequence number")
	}
	if writeHeavySeqNum != 0 && writeHeavySeqNum != boltSeqNum {
		return 0, fmt.Errorf("bolt and rocksdb numbers mismatch: %d vs %d", boltSeqNum, writeHeavySeqNum)
	}

	// It is possible we failed in the middle of a migration.  In which case the Postgres number will be
	// higher than the legacy sequence number and we should return that to start where we left off.
	if currentSeqNumPostgres > boltSeqNum {
		return currentSeqNumPostgres, nil
	}
	return boltSeqNum, nil
}

func updateRocksDB(db *gorocksdb.DB, versionBytes []byte) error {
	writeOpts := gorocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()
	if err := db.Put(writeOpts, versionBucketName, versionBytes); err != nil {
		return errors.Wrap(err, "updating version in rocksdb")
	}
	return nil
}

func updateVersion(databases *types.Databases, newVersion *storage.Version) error {
	// If the sequence number is higher than the sequence number without postgres then
	// we are migrating postgres and as such need to update the Postgres version.
	if int(newVersion.GetSeqNum()) > migrations.LastRocksDBVersionSeqNum() {
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			version.SetVersionGormDB(ctx, databases.GormDB, newVersion, false)
			return nil
		}
		return fmt.Errorf("running migration that rocks does not support: %d", newVersion.GetSeqNum())
	}

	versionBytes, err := proto.Marshal(newVersion)
	if err != nil {
		return errors.Wrap(err, "marshalling version")
	}

	err = databases.BoltDB.Update(func(tx *bolt.Tx) error {
		versionBucket, err := tx.CreateBucketIfNotExists(versionBucketName)
		if err != nil {
			return err
		}
		return versionBucket.Put(versionKey, versionBytes)
	})
	if err != nil {
		return errors.Wrap(err, "updating version in bolt")
	}

	if err := updateRocksDB(databases.RocksDB, versionBytes); err != nil {
		return err
	}

	return nil
}
