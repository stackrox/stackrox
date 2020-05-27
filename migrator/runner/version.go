package runner

import (
	"fmt"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/env"
	"github.com/tecbot/gorocksdb"
)

var (
	versionBucketName = []byte("version")
	versionKey        = []byte("\x00")
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

// getCurrentSeqNumBolt returns the current seq-num found in the bolt DB.
// A returned value of 0 means that no version information was found in the DB;
// this special value is only returned when we're upgrading from a version not using badger.
func getCurrentSeqNumBadger(db *badger.DB) (int, error) {
	var badgerVersion *storage.Version
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(versionBucketName)
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		badgerVersion = new(storage.Version)
		return item.Value(func(val []byte) error {
			return proto.Unmarshal(val, badgerVersion)
		})
	})
	if err != nil {
		return 0, errors.Wrap(err, "reading badger version")
	}

	return int(badgerVersion.GetSeqNum()), nil
}

// GetCurrentSeqNumRocksDB returns the current seq-num found in the rocks DB.
// A returned value of 0 means that no version information was found in the DB;
// this special value is only returned when we're upgrading from a version not using badger.
func GetCurrentSeqNumRocksDB(db *gorocksdb.DB) (int, error) {
	var version storage.Version

	opts := gorocksdb.NewDefaultReadOptions()
	defer opts.Destroy()
	slice, err := db.Get(opts, versionBucketName)
	if err != nil || !slice.Exists() {
		return 0, err
	}
	if err := proto.Unmarshal(slice.Data(), &version); err != nil {
		return 0, err
	}
	return int(version.GetSeqNum()), nil
}

func getCurrentSeqNum(databases *types.Databases) (int, error) {
	boltSeqNum, err := getCurrentSeqNumBolt(databases.BoltDB)
	if err != nil {
		return 0, errors.Wrap(err, "getting current bolt sequence number")
	}

	var writeHeavySeqNum int
	var writeHeavyDBName string
	if env.RocksDB.BooleanSetting() {
		writeHeavySeqNum, err = GetCurrentSeqNumRocksDB(databases.RocksDB)
		if err != nil {
			return 0, errors.Wrap(err, "getting current rocksdb sequence number")
		}
		writeHeavyDBName = "rocksdb"
	}
	if writeHeavySeqNum == 0 {
		writeHeavySeqNum, err = getCurrentSeqNumBadger(databases.BadgerDB)
		if err != nil {
			return 0, errors.Wrap(err, "getting current badger sequence number")
		}
		writeHeavyDBName = "badgerdb"
	}

	if writeHeavySeqNum != 0 && writeHeavySeqNum != boltSeqNum {
		return 0, fmt.Errorf("bolt and %s numbers mismatch: %d vs %d", writeHeavyDBName, boltSeqNum, writeHeavySeqNum)
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

	// If BadgerDB != nil, then we're going to migrate to it at the end of this function so we should update the version
	// number until the migration
	if env.RocksDB.BooleanSetting() && databases.BadgerDB == nil {
		if err := updateRocksDB(databases.RocksDB, versionBytes); err != nil {
			return err
		}
	} else {
		err = databases.BadgerDB.Update(func(txn *badger.Txn) error {
			return txn.Set(versionBucketName, versionBytes)
		})
		if err != nil {
			return errors.Wrap(err, "updating version in badger")
		}
	}

	return nil
}
