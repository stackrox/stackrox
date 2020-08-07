package singletonstore

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/bolthelper"
	"go.etcd.io/bbolt"
)

// SingletonStore is a store that stores exactly one value.
type SingletonStore interface {
	// Upsert upserts the value in the store.
	Upsert(val proto.Message) error
	// Create creates the value in the store.  Returns an error if the value already exists.
	Create(val proto.Message) error
	// Get returns the value in the store. If it doesn't exist, it returns nil, nil.
	Get() (proto.Message, error)
}

// New returns a new singleton store.
// The objectName is used for
func New(db *bbolt.DB, bucketName []byte, allocFunc func() proto.Message, objectName string) SingletonStore {
	bolthelper.RegisterBucketOrPanic(db, bucketName)
	return &singletonStore{
		bucketRef:  bolthelper.TopLevelRef(db, bucketName),
		objectName: objectName,
		allocFunc:  allocFunc,
	}
}
