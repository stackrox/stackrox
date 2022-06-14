package store

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
)

var key = []byte("\x00")

type storeImpl struct {
	bucketRef bolthelper.BucketRef
	rocksDB   *rocksdb.RocksDB
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

func (s *storeImpl) getRocksDBVersion() (*storage.Version, error) {
	if err := s.rocksDB.IncRocksDBInProgressOps(); err != nil {
		return nil, err
	}
	defer s.rocksDB.DecRocksDBInProgressOps()

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

	writeHeavyVersion, err := s.getRocksDBVersion()
	if err != nil {
		return nil, err
	}

	commonVersion := boltVersion
	if commonVersion == nil && writeHeavyVersion != nil {
		return nil, fmt.Errorf("bolt database has no version, but rocksdb does (%+v); this is invalid", writeHeavyVersion)
	}
	if writeHeavyVersion.GetSeqNum() != commonVersion.GetSeqNum() {
		return nil, fmt.Errorf("rocksdb database version mismatch: %+v vs %+v", writeHeavyVersion, commonVersion)
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

	if err := s.rocksDB.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer s.rocksDB.DecRocksDBInProgressOps()

	writeOpts := gorocksdb.NewDefaultWriteOptions()
	// Purposefully sync this
	writeOpts.SetSync(true)
	defer writeOpts.Destroy()

	if err := s.rocksDB.Put(writeOpts, versionBucket, bytes); err != nil {
		return errors.Wrap(err, "updating version in rocksdb")
	}
	return nil
}
