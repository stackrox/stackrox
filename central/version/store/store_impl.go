package store

import (
	"fmt"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var key = []byte("\x00")

type storeImpl struct {
	bucketRef bolthelper.BucketRef
}

func (s *storeImpl) GetVersion() (*storage.Version, error) {
	var version *storage.Version
	err := s.bucketRef.View(func(b *bolt.Bucket) error {
		val := b.Get(key)
		if val == nil {
			return nil
		}
		version = new(storage.Version)
		if err := proto.Unmarshal(val, version); err != nil {
			return fmt.Errorf("proto unmarshaling: %s", err)
		}
		return nil
	})
	return version, err
}

func (s *storeImpl) UpdateVersion(version *storage.Version) error {
	bytes, err := proto.Marshal(version)
	if err != nil {
		return fmt.Errorf("marshaling version %+v to proto: %s", version, err)
	}
	return s.bucketRef.Update(func(b *bolt.Bucket) error {
		if err := b.Put(key, bytes); err != nil {
			return fmt.Errorf("failed to insert: %s", err)
		}
		return nil
	})
}
