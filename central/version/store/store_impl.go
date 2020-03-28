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
)

var key = []byte("\x00")

type storeImpl struct {
	bucketRef bolthelper.BucketRef
	badgerDB  *badger.DB
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

func (s *storeImpl) GetVersion() (*storage.Version, error) {
	boltVersion, err := s.getBoltVersion()
	if err != nil {
		return nil, err
	}
	badgerVersion, err := s.getBadgerVersion()
	if err != nil {
		return nil, err
	}

	commonVersion := boltVersion
	if commonVersion == nil && badgerVersion != nil {
		return nil, fmt.Errorf("bolt database has no version, but badger does (%+v); this is invalid", badgerVersion)
	}
	if badgerVersion.GetSeqNum() != commonVersion.GetSeqNum() {
		return nil, fmt.Errorf("badger database version mismatch: %+v vs %+v", badgerVersion, commonVersion)
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

	err = s.badgerDB.Update(func(txn *badger.Txn) error {
		return txn.Set(versionBucket, bytes)
	})
	if err != nil {
		return errors.Wrap(err, "updating version in badger")
	}
	return nil
}
