package bolt

import (
	"github.com/stackrox/stackrox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var imageIntegrationBucket = []byte("imageintegrations")

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) *storeImpl {
	bolthelper.RegisterBucketOrPanic(db, imageIntegrationBucket)
	si := &storeImpl{
		DB: db,
	}
	return si
}
