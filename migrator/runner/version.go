package runner

import (
	"errors"
	"fmt"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
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
		return 0, fmt.Errorf("checking for version bucket existence: %v", err)
	}
	if !bucketExists {
		return 0, nil
	}
	versionBucket := bolthelpers.TopLevelRef(db, versionBucketName)
	versionBytes, err := bolthelpers.RetrieveElementAtKey(versionBucket, versionKey)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve version: %v", err)
	}
	if versionBytes == nil {
		return 0, errors.New("INVALID STATE: a version bucket existed, but no version was found")
	}
	version := new(storage.Version)
	err = proto.Unmarshal(versionBytes, version)
	if err != nil {
		return 0, fmt.Errorf("unmarshaling version proto: %v", err)
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
		return 0, fmt.Errorf("reading badger version: %v", err)
	}

	return int(badgerVersion.GetSeqNum()), nil
}

func getCurrentSeqNum(boltDB *bolt.DB, badgerDB *badger.DB) (int, error) {
	boltSeqNum, err := getCurrentSeqNumBolt(boltDB)
	if err != nil {
		return 0, err
	}

	badgerSeqNum, err := getCurrentSeqNumBadger(badgerDB)
	if err != nil {
		return 0, err
	}

	if badgerSeqNum != 0 && badgerSeqNum != boltSeqNum {
		return 0, fmt.Errorf("bolt and badger sequence numbers mismatch: %d vs %d", boltSeqNum, badgerSeqNum)
	}

	return boltSeqNum, nil
}

func updateVersion(boltDB *bolt.DB, badgerDB *badger.DB, newVersion *storage.Version) error {
	versionBytes, err := proto.Marshal(newVersion)
	if err != nil {
		return fmt.Errorf("marshalling version: %v", err)
	}

	err = boltDB.Update(func(tx *bolt.Tx) error {
		versionBucket, err := tx.CreateBucketIfNotExists(versionBucketName)
		if err != nil {
			return err
		}
		return versionBucket.Put(versionKey, versionBytes)
	})
	if err != nil {
		return fmt.Errorf("updating version in bolt: %v", err)
	}

	err = badgerDB.Update(func(txn *badger.Txn) error {
		return txn.Set(versionBucketName, versionBytes)
	})
	if err != nil {
		return fmt.Errorf("updating version in badger: %v", err)
	}

	return nil
}
