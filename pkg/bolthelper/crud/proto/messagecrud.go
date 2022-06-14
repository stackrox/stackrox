package proto

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/bolthelper/crud/generic"
	"github.com/stackrox/stackrox/pkg/storecache"
	bolt "go.etcd.io/bbolt"
)

// MessageCrud provides a simple crud layer on top of bolt DB for protobuf messages with a string Id field.
type MessageCrud interface {
	Read(id string) (proto.Message, error)
	ReadBatch(ids []string) ([]proto.Message, []int, error)
	ReadAll() ([]proto.Message, error)
	Count() (int, error)

	Create(msg proto.Message) error
	CreateBatch(msgs []proto.Message) error
	Update(msg proto.Message) (uint64, uint64, error)
	UpdateBatch(msgs []proto.Message) (uint64, uint64, error)
	Upsert(msg proto.Message) (uint64, uint64, error)
	UpsertBatch(msgs []proto.Message) (uint64, uint64, error)

	Delete(id string) (uint64, uint64, error)
	DeleteBatch(ids []string) (uint64, uint64, error)

	KeyFunc(msg proto.Message) []byte
}

// NewMessageCrud returns a new MessageCrud instance for the given db and bucket.
func NewMessageCrud(db *bolt.DB,
	bucketName []byte,
	keyFunc func(proto.Message) []byte,
	allocFunc func() proto.Message) (MessageCrud, error) {
	if err := bolthelper.RegisterBucket(db, bucketName); err != nil {
		return nil, err
	}
	return NewMessageCrudForBucket(bolthelper.TopLevelRef(db, bucketName), keyFunc, allocFunc), nil
}

// NewMessageCrudOrPanic returns a new MessageCrud instance for the given db and bucket or panics if the bucket can not be registered
func NewMessageCrudOrPanic(db *bolt.DB,
	bucketName []byte,
	keyFunc func(proto.Message) []byte,
	allocFunc func() proto.Message) MessageCrud {
	bolthelper.RegisterBucketOrPanic(db, bucketName)
	return NewMessageCrudForBucket(bolthelper.TopLevelRef(db, bucketName), keyFunc, allocFunc)
}

// NewCachedMessageCrud returns a new MessageCrud instance for the given db and bucket using the provided cache.
func NewCachedMessageCrud(messageCrud MessageCrud,
	cache storecache.Cache,
	metricType string,
	metricFunc func(string, string)) MessageCrud {
	return &cachedMessageCrudImpl{
		messageCrud: messageCrud,
		metricType:  metricType,
		metricFunc:  metricFunc,
		cache:       cache,
	}
}

// NewMessageCrudForBucket returns a new MessageCrud instance for the given bucket ref.
func NewMessageCrudForBucket(
	bucketRef bolthelper.BucketRef,
	keyFunc func(proto.Message) []byte,
	allocFunc func() proto.Message) MessageCrud {
	deserializeFunc := func(_, bytes []byte) (interface{}, error) {
		msg := allocFunc()
		err := proto.Unmarshal(bytes, msg)
		if err != nil {
			return nil, err
		}
		return msg, nil
	}
	serializeFunc := func(x interface{}) ([]byte, []byte, error) {
		msg := x.(proto.Message)
		key := keyFunc(msg)
		bytes, err := proto.Marshal(msg)
		return key, bytes, err
	}

	genericCrud := generic.NewCrud(bucketRef, deserializeFunc, serializeFunc)
	return &messageCrudImpl{
		genericCrud: genericCrud,
		keyFunc:     keyFunc,
	}
}
