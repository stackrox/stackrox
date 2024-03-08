package singletonstore

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/protocompat"
	"go.etcd.io/bbolt"
)

var (
	singletonKey = []byte("\x00")
)

type singletonStore struct {
	// Used for error messages
	objectName string
	allocFunc  func() protocompat.Message
	bucketRef  bolthelper.BucketRef
}

func (s *singletonStore) Upsert(val protocompat.Message) error {
	marshalled, err := proto.Marshal(val)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %s", s.objectName)
	}
	return s.bucketRef.Update(func(b *bbolt.Bucket) error {
		return b.Put(singletonKey, marshalled)
	})
}

func (s *singletonStore) Create(val protocompat.Message) error {
	marshalled, err := proto.Marshal(val)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %s", s.objectName)
	}
	return s.bucketRef.Update(func(b *bbolt.Bucket) error {
		if b.Get(singletonKey) != nil {
			return fmt.Errorf("entry with key %s already exists", singletonKey)
		}
		return b.Put(singletonKey, marshalled)
	})
}

func (s *singletonStore) Get() (protocompat.Message, error) {
	var object protocompat.Message
	err := s.bucketRef.View(func(b *bbolt.Bucket) error {
		bytes := b.Get(singletonKey)
		if bytes == nil {
			return nil
		}
		object = s.allocFunc()
		if err := protocompat.Unmarshal(bytes, object); err != nil {
			return errors.Wrapf(err, "failed to unmarshal %s", s.objectName)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return object, nil
}
