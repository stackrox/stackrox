// This file was originally generated with
// //go:generate cp ../../../../central/imageintegration/store/bolt/store.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
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

// Store provides storage functionality for alerts.
type Store interface {
	GetAll(ctx context.Context) ([]*storage.ImageIntegration, error)
	Upsert(ctx context.Context, integration *storage.ImageIntegration) error
}
