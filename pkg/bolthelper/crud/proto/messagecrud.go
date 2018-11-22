package proto

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/bolthelper/crud/generic"
)

// MessageCrud provides a simple crud layer on top of bolt DB for protobuf messages with a string Id field.
type MessageCrud interface {
	Read(id string) (proto.Message, error)
	ReadBatch(ids []string) ([]proto.Message, error)
	ReadAll() ([]proto.Message, error)
	Count() (int, error)

	Create(msg proto.Message) error
	CreateBatch(msgs []proto.Message) error
	Update(msg proto.Message) error
	UpdateBatch(msgs []proto.Message) error
	Upsert(msg proto.Message) error
	UpsertBatch(msgs []proto.Message) error

	Delete(id string) error
	DeleteBatch(ids []string) error
}

// NewMessageCrud returns a new MessageCrud instance for the given db and bucket.
func NewMessageCrud(db *bolt.DB,
	bucketName string,
	keyFunc func(proto.Message) []byte,
	allocFunc func() proto.Message) MessageCrud {
	return NewMessageCrudForBucket(bolthelper.TopLevelRef(db, []byte(bucketName)), keyFunc, allocFunc)
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
	}
}
