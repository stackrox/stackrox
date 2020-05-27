package store

import (
	"fmt"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/env"
	"github.com/tecbot/gorocksdb"
)

var key = []byte("\x00")

type storeImpl struct {
	bucketRef bolthelper.BucketRef
	badgerDB  *badger.DB
	rocksDB   *gorocksdb.DB
}

func (s *storeImpl) getBoltVersion() (*storage.Version, error) {
	var boltVersion *storage.Version
	err := s.bucketRef.View(func(b *bolt.Bucket) error {
		val := b.Get(key)
		if val == nil {
			return nil
		}
		boltVersion = new(storage.Version)
		if err := proto.Unmarshal(val, boltVersion); err != nil {
			return errors.Wrap(err, "proto unmarshaling")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return boltVersion, nil
}

func (s *storeImpl) getBadgerVersion() (*storage.Version, error) {
	var badgerVersion *storage.Version
	err := s.badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(versionBucket)
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		badgerVersion = new(storage.Version)
		return badgerhelper.UnmarshalProtoValue(item, badgerVersion)
	})
	if err != nil {
		return nil, err
	}
	return badgerVersion, nil
}

func (s *storeImpl) getRocksDBVersion() (*storage.Version, error) {
	readOpt := gorocksdb.NewDefaultReadOptions()
	defer readOpt.Destroy()

	slice, err := s.rocksDB.Get(readOpt, versionBucket)
	if err != nil || !slice.Exists() {
		return nil, err
	}
	defer slice.Free()

	var version storage.Version
	if err := proto.Unmarshal(slice.Data(), &version); err != nil {
		return nil, errors.Wrap(err, "unmarshalling versino")
	}
	return &version, nil
}

func (s *storeImpl) GetVersion() (*storage.Version, error) {
	boltVersion, err := s.getBoltVersion()
	if err != nil {
		return nil, err
	}

	var writeHeavyVersion *storage.Version
	var writeHeavyDBName string
	if env.RocksDB.BooleanSetting() {
		writeHeavyVersion, err = s.getRocksDBVersion()
		if err != nil {
			return nil, err
		}
		writeHeavyDBName = "rocksdb"
	} else {
		writeHeavyVersion, err = s.getBadgerVersion()
		if err != nil {
			return nil, err
		}
		writeHeavyDBName = "badgerdb"
	}

	commonVersion := boltVersion
	if commonVersion == nil && writeHeavyVersion != nil {
		return nil, fmt.Errorf("bolt database has no version, but %s does (%+v); this is invalid", writeHeavyDBName, writeHeavyVersion)
	}
	if writeHeavyVersion.GetSeqNum() != commonVersion.GetSeqNum() {
		return nil, fmt.Errorf("%s database version mismatch: %+v vs %+v", writeHeavyDBName, writeHeavyVersion, commonVersion)
	}

	return commonVersion, err
}

func (s *storeImpl) UpdateVersion(version *storage.Version) error {
	bytes, err := proto.Marshal(version)
	if err != nil {
		return errors.Wrapf(err, "marshaling version %+v to proto", version)
	}
	err = s.bucketRef.Update(func(b *bolt.Bucket) error {
		if err := b.Put(key, bytes); err != nil {
			return errors.Wrap(err, "failed to insert")
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "updating version in bolt")
	}

	if env.RocksDB.BooleanSetting() {
		writeOpts := gorocksdb.NewDefaultWriteOptions()
		// Purposefully sync this
		writeOpts.SetSync(true)
		defer writeOpts.Destroy()

		if err := s.rocksDB.Put(writeOpts, versionBucket, bytes); err != nil {
			return errors.Wrap(err, "updating version in rocksdb")
		}
	} else {
		err = s.badgerDB.Update(func(txn *badger.Txn) error {
			return txn.Set(versionBucket, bytes)
		})
		if err != nil {
			return errors.Wrap(err, "updating version in badger")
		}
	}
	return nil
}
